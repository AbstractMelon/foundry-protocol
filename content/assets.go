package content

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Reads image files (png/jpg/webp) from dir and returns them as data URLs
// keyed by lowercased basename, so content yaml can reference a texture by any
// case (e.g. "Iron.png" and "iron.png" both work). A missing directory yields
// an empty map: anything the content references then simply falls back to its
// color on the client.
func LoadAssets(dir string) (map[string]string, error) {
	assets := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return assets, nil
		}
		return nil, fmt.Errorf("read assets dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		mime := ""
		switch ext {
		case ".png":
			mime = "image/png"
		case ".jpg", ".jpeg":
			mime = "image/jpeg"
		case ".webp":
			mime = "image/webp"
		default:
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read asset %s: %w", name, err)
		}
		assets[strings.ToLower(name)] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	}
	return assets, nil
}

// Normalizes a texture reference from content yaml to the key used by
// LoadAssets (lowercased basename).
func TextureRefKey(ref string) string {
	return strings.ToLower(filepath.Base(ref))
}
