package world

import (
	"math"
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

func TestBeltBroadcastsProgressEveryTick(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.placeDir(t, p.ID, "belt", 0, 0, DirEast)
	belt := w.EntityAt(0, 0)
	belt.BeltItems = []BeltItem{{Res: "copper", Progress: 0.0}}
	w.TakeChanges()

	w.Tick()
	if got := belt.BeltItems[0].Progress; got <= 0.0 || got >= 1.0 {
		t.Fatalf("mid-belt item should advance every tick, got progress %v", got)
	}
	ch := w.TakeChanges()
	found := false
	for _, e := range ch.EntitiesChanged {
		if e.ID == belt.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("belt that moved items without a completion was not broadcast")
	}
}

func TestBeltCarriesProgressAcrossSegments(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.placeDir(t, p.ID, "belt", 0, 0, DirEast)
	w.placeDir(t, p.ID, "belt", 1, 0, DirEast)
	a := w.EntityAt(0, 0)
	b := w.EntityAt(1, 0)
	// An item almost finished with segment a keeps its leftover progress when
	// it slides onto segment b instead of snapping back to the start.
	a.BeltItems = []BeltItem{{Res: "copper", Progress: 0.9}}
	w.TakeChanges()

	w.Tick()
	if len(a.BeltItems) != 0 || len(b.BeltItems) != 1 {
		t.Fatalf("expected transfer to next belt, a=%v b=%v", a.BeltItems, b.BeltItems)
	}
	// The leftover progress (0.025) must survive the seam. Whether the
	// receiving belt also advanced it this same tick depends on entity
	// iteration order, so accept either.
	got := b.BeltItems[0].Progress
	if !(math.Abs(got-0.025) < 1e-9 || math.Abs(got-0.15) < 1e-9) {
		t.Fatalf("item should carry its leftover progress past the seam, got %v (want ~0.025 or ~0.15)", got)
	}
}

func TestBeltItemKeepsStableIDAcrossTransfer(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.placeDir(t, p.ID, "belt", 0, 0, DirEast)
	w.placeDir(t, p.ID, "belt", 1, 0, DirEast)
	a := w.EntityAt(0, 0)
	b := w.EntityAt(1, 0)
	// A freshly spawned item gets a non-zero stable id.
	w.addBeltItem(a, "copper")
	firstID := a.BeltItems[0].ID
	if firstID == 0 {
		t.Fatalf("freshly spawned belt item should get a non-zero id")
	}
	a.BeltItems[0].Progress = 0.9
	w.TakeChanges()

	w.Tick()
	if len(b.BeltItems) != 1 {
		t.Fatalf("expected transfer to next belt, a=%v b=%v", a.BeltItems, b.BeltItems)
	}
	if b.BeltItems[0].ID != firstID {
		t.Fatalf("item id should survive the belt transfer, want %d got %d", firstID, b.BeltItems[0].ID)
	}
	// Newly added items must never reuse an id.
	w.addBeltItem(b, "copper")
	if b.BeltItems[len(b.BeltItems)-1].ID <= firstID {
		t.Fatalf("new item ids must be strictly increasing, %v", b.BeltItems)
	}
}

func TestBeltItemMigrationAssignsIDs(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	// Simulate a pre-id save: items carry no id and there is no saved counter.
	w.FromData(&WorldData{
		NextID: 1,
		Entities: []EntityData{
			{ID: 1, Type: "belt", BeltItems: []BeltItemData{
				{Res: "copper", Progress: 0.2},
				{Res: "copper", Progress: 0.7},
			}},
		},
	})
	belt := w.entities[1]
	if len(belt.BeltItems) != 2 {
		t.Fatalf("expected 2 belt items, got %d", len(belt.BeltItems))
	}
	seen := map[int64]bool{}
	for _, bi := range belt.BeltItems {
		if bi.ID == 0 {
			t.Fatalf("migrated item must get a non-zero id")
		}
		if seen[bi.ID] {
			t.Fatalf("migrated item ids must be unique, duplicate %d", bi.ID)
		}
		seen[bi.ID] = true
	}
	// New items minted later must not collide with migrated ones.
	w.addBeltItem(belt, "copper")
	if seen[w.nextItemID] {
		t.Fatalf("newly spawned id %d collides with migrated ids", w.nextItemID)
	}
}

func TestBeltHoldsItemAtExitWhenBackedUp(t *testing.T) {
	reg := loadRegistry(t)
	w := New(reg)
	p := w.AddPlayer("p1", "Dev")
	for _, id := range reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	w.placeDir(t, p.ID, "belt", 0, 0, DirEast)
	w.placeDir(t, p.ID, "belt", 1, 0, DirEast)
	a := w.EntityAt(0, 0)
	b := w.EntityAt(1, 0)
	for i := 0; i < MaxBeltBuffer; i++ {
		b.BeltItems = append(b.BeltItems, BeltItem{Res: "copper", Progress: 0.5})
	}
	a.BeltItems = []BeltItem{{Res: "copper", Progress: 0.9}}
	w.TakeChanges()

	w.Tick()
	if len(a.BeltItems) != 1 {
		t.Fatalf("backed-up item should stay on the source belt, got %d items", len(a.BeltItems))
	}
	if got := a.BeltItems[0].Progress; got < 0.99 {
		t.Fatalf("backed-up item should wait at the belt exit, got progress %v", got)
	}
}
