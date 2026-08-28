package gateway

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr        string `yaml:"addr"`
	ServersFile string `yaml:"servers_file"`
	APIKey      string `yaml:"api_key"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{Addr: ":8080", ServersFile: "gateway/servers.yaml"}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read gateway config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse gateway config %s: %w", path, err)
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.ServersFile == "" {
		cfg.ServersFile = "gateway/servers.yaml"
	}
	return cfg, nil
}
