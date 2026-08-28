package world

import "foundryprotocol/protocol"

const (
	TileSize = 48
)

type Coord struct {
	X int
	Y int
}

// Stores the base terrain layer plus any resource deposit sitting on top of
// it. Mining the deposit away leaves the base terrain untouched.
type Tile struct {
	Terrain string
	Deposit string
	Yield   int
}

func (w *World) InBounds(x, y int) bool {
	return x >= -w.size && x <= w.size && y >= -w.size && y <= w.size
}

func (w *World) TerrainAt(x, y int) Tile {
	if t, ok := w.tiles[Coord{X: x, Y: y}]; ok {
		return t
	}
	return Tile{Terrain: w.registry.DefaultTerrain()}
}

func (w *World) TerrainIDAt(x, y int) string {
	return w.TerrainAt(x, y).Terrain
}

func (w *World) DepositAt(x, y int) (string, bool) {
	t := w.TerrainAt(x, y)
	return t.Deposit, t.Deposit != ""
}

func (w *World) SetTerrain(x, y int, id string, yield int) {
	w.tiles[Coord{X: x, Y: y}] = Tile{Terrain: id, Yield: yield}
	w.changedTiles[Coord{X: x, Y: y}] = true
}

// Places a resource deposit on top of the given base terrain.
func (w *World) SetDeposit(x, y int, terrain, res string, yield int) {
	w.tiles[Coord{X: x, Y: y}] = Tile{Terrain: terrain, Deposit: res, Yield: yield}
	w.changedTiles[Coord{X: x, Y: y}] = true
}

// Updates a tile's remaining deposit quantity. At zero the deposit is removed
// entirely so just the underlying terrain remains.
func (w *World) SetYield(x, y int, yield int) {
	t := w.TerrainAt(x, y)
	t.Yield = yield
	if t.Yield <= 0 {
		t.Deposit = ""
	}
	w.tiles[Coord{X: x, Y: y}] = t
	w.changedTiles[Coord{X: x, Y: y}] = true
}

func (w *World) BuildableAt(x, y int) bool {
	if !w.InBounds(x, y) {
		return false
	}
	t := w.TerrainAt(x, y)
	def, ok := w.registry.Terrains[t.Terrain]
	if !ok {
		return false
	}
	return def.Buildable
}

func (w *World) TileCount() int {
	return len(w.tiles)
}

func (w *World) TileViewRect(x0, y0, x1, y1 int) []protocol.TileView {
	var views []protocol.TileView
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			t := w.TerrainAt(x, y)
			views = append(views, protocol.TileView{X: x, Y: y, Terrain: t.Terrain, Deposit: t.Deposit, Yield: t.Yield})
		}
	}
	return views
}
