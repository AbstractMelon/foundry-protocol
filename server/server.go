package server

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"foundryprotocol/content"
	"foundryprotocol/protocol"
	"foundryprotocol/world"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func hashSeed(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// btoi converts a bool into an int for passing through variadic params.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

type Server struct {
	cfg        Config
	reg        *content.Registry
	assets     map[string]string
	bundle     protocol.ContentBundle
	world      *world.World
	store      *DiskStore
	logger     zerolog.Logger
	listener   net.Listener
	httpServer *http.Server

	mu       sync.Mutex
	sessions map[*Session]struct{}

	lastSnap     atomic.Value
	nextPlayerID int64
}

func New(cfg Config, logger zerolog.Logger) (*Server, error) {
	cfg = cfg.withDefaults()
	reg, err := content.LoadDir(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("content: %w", err)
	}
	assets, err := content.LoadAssets(cfg.AssetDir)
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}
	bundle := protocol.BuildContentBundle(reg, assets)
	for _, ref := range reg.ReferencedTextures() {
		if _, ok := assets[content.TextureRefKey(ref)]; !ok {
			logger.Warn().Str("texture", ref).Str("dir", cfg.AssetDir).
				Msg("content references missing texture, client will fall back to color")
		}
	}
	store := &DiskStore{Dir: cfg.SaveDir, Codec: JSONCodec{}}
	w := world.New(reg)

	seed := cfg.WorldSeed
	if seed == 0 {
		seed = int64(uint64(hashSeed(cfg.WorldName)) >> 1)
	}

	if cfg.Dev {
		w.Generate(seed)
		devSeed(w, cfg)
	} else {
		data, err := store.Load(cfg.WorldName)
		switch {
		case err == nil:
			w.FromData(data)
			if w.TileCount() == 0 {
				logger.Info().Str("world", cfg.WorldName).Msg("no terrain in save, regenerating terrain")
				w.Generate(seed)
			}
			elapsed := time.Since(data.SavedAt)
			if elapsed > 0 {
				ticks := int64(elapsed.Seconds() * float64(cfg.TPS))
				if ticks > world.MaxCatchUpTicks {
					ticks = world.MaxCatchUpTicks
				}
				w.CatchUpProduction(ticks)
				logger.Info().
					Int64("offline_seconds", int64(elapsed.Seconds())).
					Int64("catchup_ticks", ticks).
					Msg("resumed world with production catchup")
			}
		case errors.Is(err, ErrNotFound):
			logger.Info().Str("world", cfg.WorldName).Msg("no save found, created fresh world")
			w.Generate(seed)
		default:
			return nil, err
		}
	}

	s := &Server{
		cfg:          cfg,
		reg:          reg,
		assets:       assets,
		bundle:       bundle,
		world:        w,
		store:        store,
		logger:       logger,
		sessions:     make(map[*Session]struct{}),
		nextPlayerID: 1,
	}
	return s, nil
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Listen() error {
	l, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	s.listener = l
	return nil
}

func (s *Server) Serve(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.httpServer = &http.Server{Handler: mux}

	s.logger.Info().
		Str("addr", s.cfg.Addr).
		Str("world", s.cfg.WorldName).
		Int("tps", s.cfg.TPS).
		Bool("dev", s.cfg.Dev).
		Int("resources", len(s.reg.Resources)).
		Int("buildings", len(s.reg.Buildings)).
		Int("recipes", len(s.reg.Recipes)).
		Msg("world server started")

	tickDone := make(chan struct{})
	go func() {
		defer close(tickDone)
		s.runLoop(ctx)
	}()

	serveErr := make(chan error, 1)
	go func() {
		err := s.httpServer.Serve(s.listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		serveErr <- nil
	}()

	select {
	case <-ctx.Done():
		<-tickDone
		s.closeAllSessions()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Warn().Err(err).Msg("http server shutdown")
		}
		return nil
	case err := <-serveErr:
		<-tickDone
		if err != nil {
			return err
		}
		return nil
	}
}

func (s *Server) runLoop(ctx context.Context) {
	interval := time.Second / time.Duration(s.cfg.TPS)
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ticks := int64(0)
	for {
		select {
		case <-ctx.Done():
			s.save()
			return
		case <-ticker.C:
			s.step()
			ticks++
			if !s.cfg.Dev && s.cfg.AutoSaveEveryTicks > 0 && ticks%s.cfg.AutoSaveEveryTicks == 0 {
				s.save()
			}
		}
	}
}

func (s *Server) step() {
	var pending []pendingMsg
	s.mu.Lock()
	for sess := range s.sessions {
		sess.drainInbox(&pending)
	}
	s.mu.Unlock()

	for _, p := range pending {
		s.applyMessage(p.sess, p.msg)
	}

	s.world.Tick()
	ch := s.world.TakeChanges()
	// s.logger.Debug().Int("added", len(ch.EntitiesAdded)).Int("removed", len(ch.EntitiesRemoved)).Int("changed", len(ch.EntitiesChanged)).Int("tiles", len(ch.TilesChanged)).Msg("tick changes")
	diffMsg := s.world.BuildDiff(ch)
	snapMsg := s.world.BuildSnapshot()

	if raw, err := snapMsg.Encode(); err == nil {
		s.lastSnap.Store(raw)
	} else {
		s.logger.Warn().Err(err).Msg("encode snapshot")
	}

	s.mu.Lock()
	for sess := range s.sessions {
		if sess.playerID == "" {
			continue
		}
		sess.enqueue(diffMsg)
	}
	s.mu.Unlock()
}

func (s *Server) applyMessage(sess *Session, msg protocol.Message) {
	if msg.Type != protocol.TypeHello && sess.playerID == "" {
		return
	}
	switch msg.Type {
	case protocol.TypeHello:
		s.handleHello(sess, msg)
	case protocol.TypePlaceBuilding:
		s.logger.Info().Str("player", sess.playerID).Str("type", msg.BuildingType).Int("x", msg.TileX).Int("y", msg.TileY).Msg("place_building received")
		if err := s.world.PlaceBuilding(sess.playerID, msg.BuildingType, msg.TileX, msg.TileY, msg.Dir, btoi(msg.Flipped)); err != nil {
			s.logger.Info().Err(err).Msg("place_building rejected")
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: err.Error()})
		} else {
			s.logger.Info().Msg("place_building accepted")
		}
	case protocol.TypeRemoveBuilding:
		if err := s.world.RemoveBuilding(sess.playerID, msg.EntityID); err != nil {
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: err.Error()})
		}
	case protocol.TypeChat:
		s.handleChat(sess, msg.Text)
	}
}

func (s *Server) handleHello(sess *Session, msg protocol.Message) {
	name := strings.TrimSpace(msg.Name)
	if name == "" {
		name = "Player"
	}
	p := s.world.PlayerByName(name)
	if p == nil {
		id := "p" + strconv.FormatInt(s.nextPlayerID, 10)
		s.nextPlayerID++
		p = s.world.AddPlayer(id, name)
		if s.cfg.Dev {
			s.grantDevResources(id)
		}
		s.logger.Info().Str("player", name).Msg("player joined")
	}
	// Give every player a material hub at their spawn (idempotent).
	s.world.EnsureHub(p.ID)
	sess.playerID = p.ID
	welcome := s.world.BuildWelcome(
		p.ID,
		p.Name,
		s.cfg.WorldName,
		s.bundle,
		s.world.Snapshot(),
	)
	sess.enqueue(welcome)
}

func (s *Server) handleChat(sess *Session, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if s.cfg.Dev && strings.HasPrefix(text, "/") {
		s.devCommand(sess, text)
		return
	}
	name := "?"
	if p := s.world.GetPlayer(sess.playerID); p != nil {
		name = p.Name
	}
	s.logger.Debug().Str("from", name).Str("text", text).Msg("broadcasting chat")
	s.broadcast(protocol.Message{
		Type:       protocol.TypeChatReceived,
		PlayerID:   sess.playerID,
		PlayerName: name,
		Text:       text,
	})
}

func (s *Server) broadcast(m protocol.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sess := range s.sessions {
		sess.enqueue(m)
	}
}

func (s *Server) save() {
	if s.cfg.Dev {
		return
	}
	data := s.world.ToData()
	data.SavedAt = time.Now()
	if err := s.store.Save(s.cfg.WorldName, &data); err != nil {
		s.logger.Err(err).Str("world", s.cfg.WorldName).Msg("save failed")
		return
	}
	s.logger.Debug().Str("world", s.cfg.WorldName).Msg("world saved")
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sess := newSession(s, conn)
	s.mu.Lock()
	s.sessions[sess] = struct{}{}
	s.mu.Unlock()
	s.logger.Info().Str("remote", conn.RemoteAddr().String()).Msg("client connected")
	go sess.writerLoop()
	go sess.readerLoop()
}

func (s *Server) removeSession(sess *Session) {
	s.mu.Lock()
	delete(s.sessions, sess)
	s.mu.Unlock()
}

func (s *Server) closeAllSessions() {
	s.mu.Lock()
	sessions := make([]*Session, 0, len(s.sessions))
	for sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()
	for _, sess := range sessions {
		sess.closeAndCleanup()
	}
}
