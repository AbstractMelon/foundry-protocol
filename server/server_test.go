package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"foundryprotocol/protocol"
)

func startTestServer(t *testing.T) *Server {
	t.Helper()
	logger := zerolog.Nop()
	srv, err := New(Config{
		Addr:       "127.0.0.1:0",
		WorldName:  "test",
		ContentDir: "../content",
		SaveDir:    t.TempDir(),
		TPS:        20,
		Dev:        true,
	}, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = srv.Serve(ctx)
	}()
	t.Cleanup(cancel)
	return srv
}

func newTestConn(t *testing.T, url, name string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(protocol.Message{Type: protocol.TypeHello, Name: name}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) protocol.Message {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg protocol.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return msg
}

func waitFor(t *testing.T, conn *websocket.Conn, wantType string, check func(protocol.Message) bool) protocol.Message {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg := readMessage(t, conn)
		if msg.Type == wantType && check(msg) {
			return msg
		}
	}
	t.Fatalf("timed out waiting for %q", wantType)
	return protocol.Message{}
}

func TestContentBundleEmbedsAvailableTextures(t *testing.T) {
	logger := zerolog.Nop()
	srv, err := New(Config{
		Addr:       "127.0.0.1:0",
		WorldName:  "test",
		ContentDir: "../content",
		AssetDir:   "../assets",
		SaveDir:    t.TempDir(),
		TPS:        20,
		Dev:        true,
	}, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	assets, err := os.ReadDir("../assets")
	if err != nil {
		t.Fatalf("read assets dir: %v", err)
	}
	want := 0
	for _, f := range assets {
		if f.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(f.Name())) {
		case ".png", ".jpg", ".jpeg", ".webp":
			want++
		}
	}
	if want == 0 {
		t.Fatal("expected at least one asset file in ../assets")
	}
	if got := len(srv.bundle.Textures); got != want {
		t.Fatalf("expected %d embedded textures, got %d (%v)", want, got, srv.bundle.Textures)
	}
}

func TestWorldServerSmoke(t *testing.T) {
	srv := startTestServer(t)
	url := "ws://" + srv.Addr() + "/ws"

	conn := newTestConn(t, url, "Tester")
	welcome := waitFor(t, conn, protocol.TypeWelcome, func(m protocol.Message) bool { return true })
	if welcome.PlayerID == "" {
		t.Fatal("welcome has no player id")
	}
	if welcome.Snapshot == nil || welcome.Content == nil {
		t.Fatal("welcome missing snapshot or content")
	}
	if len(welcome.Content.Buildings) == 0 || len(welcome.Content.Resources) == 0 {
		t.Fatal("content bundle empty")
	}

	if err := conn.WriteJSON(protocol.Message{
		Type:         protocol.TypePlaceBuilding,
		BuildingType: "miner",
		TileX:        10,
		TileY:        10,
	}); err != nil {
		t.Fatalf("place: %v", err)
	}

	got := waitFor(t, conn, protocol.TypeDiff, func(m protocol.Message) bool {
		return m.Diff != nil && len(m.Diff.EntitiesAdded) == 1 && m.Diff.EntitiesAdded[0].Type == "miner"
	})
	minerID := got.Diff.EntitiesAdded[0].ID

	if err := conn.WriteJSON(protocol.Message{Type: protocol.TypeRemoveBuilding, EntityID: minerID}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	waitFor(t, conn, protocol.TypeDiff, func(m protocol.Message) bool {
		return m.Diff != nil && len(m.Diff.EntitiesRemoved) == 1 && m.Diff.EntitiesRemoved[0] == minerID
	})
}

func TestDevChatCommands(t *testing.T) {
	srv := startTestServer(t)
	url := "ws://" + srv.Addr() + "/ws"
	conn := newTestConn(t, url, "Wizard")

	waitFor(t, conn, protocol.TypeWelcome, func(m protocol.Message) bool { return true })
	if err := conn.WriteJSON(protocol.Message{Type: protocol.TypeChat, Text: "/give copper 500"}); err != nil {
		t.Fatalf("chat: %v", err)
	}

	waitFor(t, conn, protocol.TypeSystem, func(m protocol.Message) bool {
		return strings.Contains(m.Text, "copper")
	})
}

func TestChatBroadcast(t *testing.T) {
	srv := startTestServer(t)
	url := "ws://" + srv.Addr() + "/ws"
	alice := newTestConn(t, url, "Alice")
	bob := newTestConn(t, url, "Bob")
	waitFor(t, alice, protocol.TypeWelcome, func(m protocol.Message) bool { return true })
	waitFor(t, bob, protocol.TypeWelcome, func(m protocol.Message) bool { return true })

	if err := alice.WriteJSON(protocol.Message{Type: protocol.TypeChat, Text: "hi there"}); err != nil {
		t.Fatalf("chat: %v", err)
	}
	got := waitFor(t, bob, protocol.TypeChatReceived, func(m protocol.Message) bool { return m.Text == "hi there" && m.PlayerName == "Alice" })
	if got.PlayerName != "Alice" {
		t.Fatalf("unexpected sender %q", got.PlayerName)
	}
}
