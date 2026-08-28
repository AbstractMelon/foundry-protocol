package gateway

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type ServerInfo struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	WSURL       string `yaml:"ws_url" json:"ws_url"`
	Owner       string `yaml:"owner" json:"owner"`
	MaxPlayers  int    `yaml:"max_players" json:"max_players"`
}

type serversFile struct {
	Servers []ServerInfo `yaml:"servers"`
}

type Registry struct {
	mu      sync.RWMutex
	servers map[string]ServerInfo
	path    string
}

func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{
		servers: make(map[string]ServerInfo),
		path:    path,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read servers file %s: %w", path, err)
	}
	var doc serversFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse servers file %s: %w", path, err)
	}
	for _, s := range doc.Servers {
		if s.ID != "" {
			r.servers[s.ID] = s
		}
	}
	return r, nil
}

func (r *Registry) List() []ServerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ServerInfo, 0, len(r.servers))
	for _, s := range r.servers {
		out = append(out, s)
	}
	return out
}

func (r *Registry) Add(info ServerInfo) error {
	if info.ID == "" || info.WSURL == "" {
		return fmt.Errorf("server id and ws_url are required")
	}
	r.mu.Lock()
	r.servers[info.ID] = info
	r.mu.Unlock()
	return r.persist()
}

func (r *Registry) persist() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc := serversFile{}
	for _, s := range r.servers {
		doc.Servers = append(doc.Servers, s)
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode servers: %w", err)
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write servers tmp file: %w", err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		return fmt.Errorf("replace servers file: %w", err)
	}
	return nil
}
