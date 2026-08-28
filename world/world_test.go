package world

import (
	"testing"

	"foundryprotocol/content"
)

func TestProductionChain(t *testing.T) {
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
	if err := w.PlaceBuilding(p.ID, "smelter", 2, 0); err != nil {
		t.Fatalf("place smelter: %v", err)
	}
	w.EntityAt(2, 0).Stock["copper"] = 100
	w.TakeChanges()

	for i := 0; i < 100; i++ {
		w.Tick()
	}
	smelter := w.EntityAt(2, 0)
	if smelter.Stock["copper_bar"] < 1 {
		t.Fatalf("smelter produced no bars: stock=%v", smelter.Stock)
	}
	if p.Resources["copper"] != 970 {
		t.Fatalf("production should not touch player inventory, copper=%d", p.Resources["copper"])
	}
}

func TestPlaceBuildingDeductions(t *testing.T) {
	reg, _ := content.LoadDir("../content")
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	p.Resources["copper"] = 35
	if err := w.PlaceBuilding(p.ID, "miner", 5, 5); err != nil {
		t.Fatalf("place: %v", err)
	}
	if p.Resources["copper"] != 5 {
		t.Fatalf("expected copper 5, got %d", p.Resources["copper"])
	}
	if w.EntityAt(5, 5) == nil {
		t.Fatal("entity not placed")
	}
	if err := w.PlaceBuilding(p.ID, "miner", 5, 5); err == nil {
		t.Fatal("expected occupied-tile error")
	}
	if err := w.PlaceBuilding(p.ID, "miner", 8, 8); err == nil {
		t.Fatal("expected insufficient-resources error")
	}
}

func TestClearAllEntities(t *testing.T) {
	reg, _ := content.LoadDir("../content")
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	for i := 0; i < 5; i++ {
		_ = w.PlaceBuilding(p.ID, "wall", i, 0)
	}
	w.TakeChanges()
	if n := w.ClearAllEntities(); n != 5 {
		t.Fatalf("cleared %d, want 5", n)
	}
	if w.EntityCount() != 0 {
		t.Fatal("entities remain after clear")
	}
}
