package world

import "time"

const SaveVersion = 2

type WorldData struct {
	Version  int          `json:"version"`
	Tick     int64        `json:"tick"`
	NextID   int64        `json:"next_entity_id"`
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
	ID       int64          `json:"id"`
	Type     string         `json:"type"`
	OwnerID  string         `json:"owner_id"`
	X        int            `json:"x"`
	Y        int            `json:"y"`
	Health   int            `json:"health"`
	Progress int            `json:"progress"`
	Dir      int            `json:"dir"`
	Stock    map[string]int `json:"stock"`
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
		data.Entities = append(data.Entities, EntityData{
			ID:       e.ID,
			Type:     e.Type,
			OwnerID:  e.OwnerID,
			X:        e.X,
			Y:        e.Y,
			Health:   e.Health,
			Progress: e.Progress,
			Dir:      e.Dir,
			Stock:    e.Stock,
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
	for _, pd := range d.Players {
		w.players[pd.ID] = &Player{ID: pd.ID, Name: pd.Name, Resources: pd.Resources}
	}
	for _, ed := range d.Entities {
		w.entities[ed.ID] = &Entity{
			ID:       ed.ID,
			Type:     ed.Type,
			OwnerID:  ed.OwnerID,
			X:        ed.X,
			Y:        ed.Y,
			Health:   ed.Health,
			Progress: ed.Progress,
			Dir:      ed.Dir,
			Stock:    ed.Stock,
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
