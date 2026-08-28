package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

type Server struct {
	cfg    Config
	reg    *Registry
	logger zerolog.Logger
}

func NewServer(cfg Config, reg *Registry, logger zerolog.Logger) *Server {
	return &Server{cfg: cfg, reg: reg, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/servers", s.handleListServers)
	mux.HandleFunc("/api/servers", s.handleRegisterServer)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"servers": s.reg.List()})
}

func (s *Server) handleRegisterServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if s.cfg.APIKey == "" || auth != "Bearer "+s.cfg.APIKey {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var info ServerInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := s.reg.Add(info); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.logger.Info().Str("server_id", info.ID).Str("ws_url", info.WSURL).Msg("world server registered")
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "server": info})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
