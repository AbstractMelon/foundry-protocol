package world

import "time"

const SaveVersion = 3

type WorldData struct {
	Version  int          `json:"version"`
	Tick     int64        `json:"tick"`
	NextID   int64        `json:"next_entity_id"`
	NextItemID int64      `json:"next_item_id"`
	Players  []PlayerData `json:"players"`
	Entities []EntityData `json:"entities"`
	Tiles    []TileData   `json:"tiles"`
	SavedAt  time.Time    `json:"saved_at,omitempty"`
}

type PlayerData struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Resources map[string]int `json:"resources"`
}

type EntityData struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	OwnerID   string         `json:"owner_id"`
	X         int            `json:"x"`
	Y         int            `json:"y"`
	Health    int            `json:"health"`
	Progress  int            `json:"progress"`
	Dir       int            `json:"dir"`
	Flipped   bool           `json:"flipped,omitempty"`
	Stock     map[string]int `json:"stock"`
	BeltItems []BeltItemData `json:"belt_items,omitempty"`
}

type BeltItemData struct {
	ID       int64   `json:"id"`
	Res      string  `json:"res"`
	Progress float64 `json:"progress"`
}

type TileData struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Terrain string `json:"terrain"`
	Deposit string `json:"deposit,omitempty"`
	Yield   int    `json:"yield,omitempty"`
}

func (w *World) ToData() WorldData {
	data := WorldData{
		Version: SaveVersion,
		Tick:    w.tick,
		NextID:  w.nextID,
		NextItemID: w.nextItemID,
	}
	for _, id := range sortedPlayerIDs(w.players) {
		p := w.players[id]
		data.Players = append(data.Players, PlayerData{
			ID:        p.ID,
			Name:      p.Name,
			Resources: p.Resources,
		})
	}
	for _, id := range sortedEntityIDs(w.entities) {
		e := w.entities[id]
		beltItems := make([]BeltItemData, len(e.BeltItems))
		for i, bi := range e.BeltItems {
			beltItems[i] = BeltItemData{ID: bi.ID, Res: bi.Res, Progress: bi.Progress}
		}
		data.Entities = append(data.Entities, EntityData{
			ID:       e.ID,
			Type:     e.Type,
			OwnerID:  e.OwnerID,
			X:        e.X,
			Y:        e.Y,
			Health:   e.Health,
			Progress: e.Progress,
			Dir:      e.Dir,
			Flipped:  e.Flipped,
			Stock:    e.Stock,
			BeltItems: beltItems,
		})
	}
	for _, c := range sortedTileKeys(w.tiles) {
		t := w.tiles[c]
		data.Tiles = append(data.Tiles, TileData{X: c.X, Y: c.Y, Terrain: t.Terrain, Deposit: t.Deposit, Yield: t.Yield})
	}
	return data
}

func (w *World) FromData(d *WorldData) {
	w.tick = d.Tick
	w.nextID = d.NextID
	w.nextItemID = d.NextItemID
	// Migrate belt items saved before they carried a stable ID (id == 0) by
	// minting fresh IDs so the client always has something distinct to track.
	migrateMissing := func(id int64) int64 {
		if id != 0 {
			return id
		}
		w.nextItemID++
		return w.nextItemID
	}
	for _, pd := range d.Players {
		w.players[pd.ID] = &Player{ID: pd.ID, Name: pd.Name, Resources: pd.Resources}
	}
	for _, ed := range d.Entities {
		beltItems := make([]BeltItem, len(ed.BeltItems))
		for i, bi := range ed.BeltItems {
			beltItems[i] = BeltItem{ID: migrateMissing(bi.ID), Res: bi.Res, Progress: bi.Progress}
		}
		w.entities[ed.ID] = &Entity{
			ID:       ed.ID,
			Type:     ed.Type,
			OwnerID:  ed.OwnerID,
			X:        ed.X,
			Y:        ed.Y,
			Health:   ed.Health,
			Progress: ed.Progress,
			Dir:      ed.Dir,
			Flipped:  ed.Flipped,
			Stock:    ed.Stock,
			BeltItems: beltItems,
		}
	}
	w.tiles = make(map[Coord]Tile)
	for _, td := range d.Tiles {
		base := td.Terrain
		deposit := td.Deposit
		if _, ok := w.registry.Terrains[td.Terrain]; !ok {
			if m, ok := legacyOreVein[td.Terrain]; ok {
				base, deposit = m.base, m.deposit
			} else {
				base = w.registry.DefaultTerrain()
			}
		}
		w.tiles[Coord{X: td.X, Y: td.Y}] = Tile{Terrain: base, Deposit: deposit, Yield: td.Yield}
	}
	w.changedTiles = make(map[Coord]bool)
}

// Maps ore-vein terrain ids from saves created before deposits became a tile
// overlay, so mining amounts survive the migration.
var legacyOreVein = map[string]struct{ base, deposit string }{
	"copper_vein": {"rock", "copper"},
	"iron_vein":   {"rock", "iron"},
	"coal_seam":   {"rock", "coal"},
}
