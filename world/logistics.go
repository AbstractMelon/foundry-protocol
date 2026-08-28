package world

// Belt movements per tick. A belt holds a small buffer of items in its Stock
// and pushes them toward the tile it faces. A hub absorbs any item it receives
// and delivers it straight to its owner's inventory.
const (
	DirNorth = 0
	DirEast  = 1
	DirSouth = 2
	DirWest  = 3

	MaxBeltBuffer = 5 // total items a single belt can hold in flight
)

func dirOffset(dir int) (dx, dy int) {
	switch dir {
	case DirNorth:
		return 0, -1
	case DirEast:
		return 1, 0
	case DirSouth:
		return 0, 1
	case DirWest:
		return -1, 0
	default:
		return 1, 0
	}
}

func (w *World) neighborEntity(e *Entity) *Entity {
	dx, dy := dirOffset(e.Dir)
	return w.EntityAt(e.X+dx, e.Y+dy)
}

func itemCount(e *Entity) int {
	total := 0
	for _, qty := range e.Stock {
		total += qty
	}
	return total
}

func takeItem(e *Entity, res string) bool {
	if e.Stock[res] <= 0 {
		return false
	}
	e.Stock[res]--
	if e.Stock[res] == 0 {
		delete(e.Stock, res)
	}
	return true
}

func addItem(e *Entity, res string) {
	e.Stock[res]++
}

// Reports how many of a resource a building can still hold in its internal
// stock (used as the acceptance limit for non-belt, non-hub targets).
func (w *World) stockCap(e *Entity, res string, storage map[string]int) int {
	if cap, ok := storage[res]; ok {
		return cap - e.Stock[res]
	}
	if _, inStock := e.Stock[res]; inStock {
		return MaxBeltBuffer
	}
	return 0
}

// Attempts to push one item of the given resource out of `from` (in its facing
// direction) into an adjacent receiver. It returns true when the item left
// the source stock.
func (w *World) receiveInto(from *Entity, res string) bool {
	if from.Stock[res] <= 0 {
		return false
	}
	target := w.neighborEntity(from)
	if target == nil {
		return false
	}
	tdef, ok := w.registry.Buildings[target.Type]
	if !ok {
		return false
	}

	// A hub immediately delivers everything to its owner's inventory.
	if target.Type == "hub" {
		if !takeItem(from, res) {
			return false
		}
		w.deliverToOwner(target, res)
		w.markChanged(from)
		return true
	}

	switch tdef.Category {
	case "logistics":
		// Another belt can accept up to its buffer limit.
		if itemCount(target) >= MaxBeltBuffer {
			return false
		}
		if !takeItem(from, res) {
			return false
		}
		addItem(target, res)
		w.markChanged(from)
		w.markChanged(target)
		return true
	default:
		// A production/other building accepts the item up to its storage cap.
		if w.stockCap(target, res, tdef.Storage) <= 0 {
			return false
		}
		if !takeItem(from, res) {
			return false
		}
		addItem(target, res)
		w.markChanged(from)
		w.markChanged(target)
		return true
	}
}

// Credits a hub's owner with `res`, marking the player changed.
func (w *World) deliverToOwner(hub *Entity, res string) {
	p := w.players[hub.OwnerID]
	if p == nil {
		return
	}
	p.Resources[res]++
	w.changedPlayers[hub.OwnerID] = true
}

// Advances every belt: one buffered item per tick moves one tile toward its
// destination so belts form a flowing line rather than dumping whole buffers
// at once.
func (w *World) tickLogistics() {
	for _, e := range w.entities {
		tdef, ok := w.registry.Buildings[e.Type]
		if !ok || tdef.Category != "logistics" {
			continue
		}
		if e.Stock == nil || itemCount(e) == 0 {
			continue
		}
		for res := range e.Stock {
			if w.receiveInto(e, res) {
				break
			}
		}
	}
}
