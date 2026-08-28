package world

import (
	"slices"
	"testing"

	"foundryprotocol/content"
)

func TestGenerateProducesTerrain(t *testing.T) {
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	w := New(reg)
	w.Generate(42)
	if w.TileCount() == 0 {
		t.Fatal("no terrain generated")
	}
	if !w.BuildableAt(0, 0) {
		t.Fatal("center should be buildable")
	}
	if w.BuildableAt(w.size+1, w.size+1) {
		t.Fatal("off-grid should not be buildable")
	}
}

func TestBuildableOnNonOreGround(t *testing.T) {
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	w := New(reg)
	w.SetTerrain(0, 0, "water", 0)
	w.SetTerrain(1, 0, "grass", 0)
	if w.BuildableAt(0, 0) {
		t.Fatal("water should not be buildable")
	}
	if !w.BuildableAt(1, 0) {
		t.Fatal("grass should be buildable")
	}
}

func TestGeneratePlacesDepositsOnAllowedTerrain(t *testing.T) {
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	w := New(reg)
	w.Generate(42)
	deposits := 0
	for x := -w.size; x <= w.size; x++ {
		for y := -w.size; y <= w.size; y++ {
			tile := w.TerrainAt(x, y)
			if tile.Deposit == "" {
				continue
			}
			deposits++
			res, ok := reg.Resources[tile.Deposit]
			if !ok {
				t.Fatalf("deposit references unknown resource %q", tile.Deposit)
			}
			if !slices.Contains(res.CanPlaceOn, tile.Terrain) {
				t.Fatalf("deposit %q sits on disallowed terrain %q", tile.Deposit, tile.Terrain)
			}
		}
	}
	if deposits == 0 {
		t.Fatal("no deposits generated")
	}
}

func TestMiningDepletesDepositLeavesTerrain(t *testing.T) {
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.SetDeposit(0, 0, "grass", "copper", 3)
	if !w.BuildableAt(0, 0) {
		t.Fatal("deposit should not make the base terrain unbuildable")
	}
	if err := w.PlaceBuilding(p.ID, "miner", 0, 0); err != nil {
		t.Fatalf("place miner on deposit: %v", err)
	}
	w.TakeChanges()

	for i := 0; i < 1000; i++ {
		w.Tick()
		if _, has := w.DepositAt(0, 0); !has {
			break
		}
	}
	tile := w.TerrainAt(0, 0)
	if _, has := w.DepositAt(0, 0); has {
		t.Fatalf("deposit should be depleted, tile=%+v", tile)
	}
	if tile.Terrain != "grass" {
		t.Fatalf("base terrain should be preserved after mining, got %q", tile.Terrain)
	}
	if miner := w.EntityAt(0, 0); miner.Stock["copper"] != 3 {
		t.Fatalf("expected 3 copper mined, got %d", miner.Stock["copper"])
	}
}

func TestMinerOnNonOreProducesNothing(t *testing.T) {
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	if err := w.PlaceBuilding(p.ID, "miner", 0, 0); err != nil {
		t.Fatalf("place miner: %v", err)
	}
	w.TakeChanges()
	for i := 0; i < 50; i++ {
		w.Tick()
	}
	if miner := w.EntityAt(0, 0); len(miner.Stock) != 0 {
		t.Fatalf("miner on grass should produce nothing, got %v", miner.Stock)
	}
}
