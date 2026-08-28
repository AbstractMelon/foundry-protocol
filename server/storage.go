package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"foundryprotocol/world"
)

var ErrNotFound = errors.New("save file not found")

type Codec interface {
	Encode(w io.Writer, data *world.WorldData) error
	Decode(r io.Reader) (*world.WorldData, error)
}

type JSONCodec struct{}

func (JSONCodec) Encode(w io.Writer, data *world.WorldData) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (JSONCodec) Decode(r io.Reader) (*world.WorldData, error) {
	var data world.WorldData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

type DiskStore struct {
	Dir   string
	Codec Codec
}

func (s *DiskStore) path(worldName string) string {
	return filepath.Join(s.Dir, worldName+".json")
}

func (s *DiskStore) Save(worldName string, data *world.WorldData) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create save dir: %w", err)
	}
	path := s.path(worldName)
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create save file: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if err := s.Codec.Encode(f, data); err != nil {
		return fmt.Errorf("encode save: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close save file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("finalize save file: %w", err)
	}
	ok = true
	return nil
}

func (s *DiskStore) Load(worldName string) (*world.WorldData, error) {
	f, err := os.Open(s.path(worldName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("open save file: %w", err)
	}
	defer f.Close()
	data, err := s.Codec.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode save file: %w", err)
	}
	return data, nil
}
