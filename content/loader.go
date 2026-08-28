package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type fileDoc struct {
	Resources []Resource    `yaml:"resources"`
	Buildings []Building    `yaml:"buildings"`
	Recipes   []Recipe      `yaml:"recipes"`
	Terrains  []TerrainType `yaml:"terrains"`
}

func LoadDir(dir string) (*Registry, error) {
	reg := New()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read content dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var doc fileDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, res := range doc.Resources {
			reg.AddResource(res)
		}
		for _, b := range doc.Buildings {
			reg.AddBuilding(b)
		}
		for _, rec := range doc.Recipes {
			reg.AddRecipe(rec)
		}
		for _, t := range doc.Terrains {
			reg.AddTerrain(t)
		}
	}
	if errs := reg.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("content validation failed: %s", strings.Join(errs, "; "))
	}
	return reg, nil
}
