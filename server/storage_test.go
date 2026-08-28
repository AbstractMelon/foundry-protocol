package server

import (
	"errors"
	"testing"
	"time"

	"foundryprotocol/world"
)

func TestDiskStoreRoundTrip(t *testing.T) {
	store := &DiskStore{Dir: t.TempDir(), Codec: JSONCodec{}}
	data := &world.WorldData{
		Version: world.SaveVersion,
		Tick:    1234,
		NextID:  7,
		Players: []world.PlayerData{
			{ID: "p1", Name: "Alice", Resources: map[string]int{"copper": 100, "iron": 50}},
		},
		Entities: []world.EntityData{
			{ID: 0, Type: "miner", OwnerID: "p1", X: 2, Y: 0, Health: 300, Progress: 5, Stock: map[string]int{"copper": 3}},
		},
		SavedAt: time.Now().Add(-time.Hour),
	}
	if err := store.Save("testworld", data); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := store.Load("testworld")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Tick != data.Tick || loaded.NextID != data.NextID {
		t.Fatalf("mismatch: %+v", loaded)
	}
	if len(loaded.Players) != 1 || loaded.Players[0].Name != "Alice" {
		t.Fatalf("players mismatch: %+v", loaded.Players)
	}
	if len(loaded.Entities) != 1 || loaded.Entities[0].Stock["copper"] != 3 {
		t.Fatalf("entities mismatch: %+v", loaded.Entities)
	}
	if _, err := store.Load("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
