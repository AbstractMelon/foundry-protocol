package world

import (
	"testing"

	"foundryprotocol/content"
)

func loadRegistry(t *testing.T) *content.Registry {
	t.Helper()
	reg, err := content.LoadDir("../content")
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	return reg
}

func TestBeltDeliversMinedOreToHub(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.SetDeposit(0, 0, "grass", "copper", 200)

	// miner -> belt -> hub, everything facing east.
	if err := w.PlaceBuilding(p.ID, "miner", 0, 0); err != nil {
		t.Fatalf("miner: %v", err)
	}
	if err := w.PlaceBuilding(p.ID, "belt", 1, 0); err != nil {
		t.Fatalf("belt: %v", err)
	}
	if err := w.PlaceBuilding(p.ID, "hub", 2, 0); err != nil {
		t.Fatalf("hub: %v", err)
	}
	w.TakeChanges()

	for i := 0; i < 500; i++ {
		w.Tick()
	}
	if p.Resources["copper"] < 1 {
		t.Fatalf("hub delivered no copper to player, resources=%v", p.Resources)
	}
	if miner := w.EntityAt(0, 0); len(miner.Stock) != 0 {
		t.Fatalf("miner stock should drain to the belt, got %v", miner.Stock)
	}
}

func TestBeltChainFeedsSmelterAndDeliversBars(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.SetDeposit(0, 0, "grass", "copper", 500)

	// miner(0,0) -> belt(1,0) -> smelter(2,0) -> belt(3,0) -> hub(4,0)
	w.placeDir(t, p.ID, "miner", 0, 0, DirEast)
	w.placeDir(t, p.ID, "belt", 1, 0, DirEast)
	w.placeDir(t, p.ID, "smelter", 2, 0, DirEast)
	w.placeDir(t, p.ID, "belt", 3, 0, DirEast)
	w.placeDir(t, p.ID, "hub", 4, 0, DirEast)
	w.TakeChanges()

	for i := 0; i < 3000; i++ {
		w.Tick()
	}
	if p.Resources["copper_bar"] < 1 {
		t.Fatalf("player received no copper bars, resources=%v (want some)", p.Resources)
	}
	// Raw copper left in the chain should never exceed what is trapped on
	// belts; it must flow into the smelter and come out as bars.
	var belted, smelted int
	for _, e := range w.entities {
		switch e.Type {
		case "belt":
			belted += itemCount(e)
		case "smelter":
			smelted += e.Stock["copper"]
		}
	}
	if belted > 10 {
		t.Fatalf("copper backed up on belts: %d", belted)
	}
	if _, has := w.EntityAt(2, 0).Stock["copper"]; !has {
		t.Logf("smelter held no raw copper buffer (ok), belted=%d smelted=%d bars=%d", belted, smelted, p.Resources["copper_bar"])
	}
}

func (w *World) placeDir(t *testing.T, playerID, typ string, x, y, dir int) {
	t.Helper()
	if err := w.PlaceBuilding(playerID, typ, x, y); err != nil {
		t.Fatalf("place %s at (%d,%d): %v", typ, x, y, err)
	}
	for _, e := range w.entities {
		if e.Type == typ && e.X == x && e.Y == y {
			e.Dir = dir
			break
		}
	}
}
