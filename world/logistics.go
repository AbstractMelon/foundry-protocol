package world

import "sort"

// Belt movements per tick. Items slowly slide along belts as individual
// tracked entities with a progress value that advances each tick.
const (
	DirNorth = 0
	DirEast  = 1
	DirSouth = 2
	DirWest  = 3

	MaxBeltBuffer = 5 // total items a single belt can hold in flight
	BeltSpeed     = 8 // ticks for an item to travel one belt segment
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

func isBelt(e *Entity) bool {
	return e != nil && (e.Type == "belt" || e.Type == "belt_turn")
}

func isBeltTurn(e *Entity) bool {
	return e != nil && e.Type == "belt_turn"
}

// beltTurnInputDir returns the absolute direction from which a corner belt
// accepts items. The turn always pivots a quarter turn from its input side to
// its output side (e.Dir); flipping mirrors the bend. For the unflipped sprite
// (input West -> output South) the input is one step past the output, and for
// the flipped sprite it is one step before.
func beltTurnInputDir(e *Entity) int {
	if e.Flipped {
		return (e.Dir + 3) % 4
	}
	return (e.Dir + 1) % 4
}

// receivesFromInputSide reports whether `src` sits on the corner belt's input
// side (i.e. items flowing out of `src` enter the bend the correct way).
func cornerAcceptsFrom(src, corner *Entity) bool {
	if !isBeltTurn(corner) {
		return true
	}
	idx, idy := dirOffset(beltTurnInputDir(corner))
	return src != nil && src.X == corner.X+idx && src.Y == corner.Y+idy
}

func (w *World) neighborEntity(e *Entity) *Entity {
	dx, dy := dirOffset(e.Dir)
	return w.EntityAt(e.X+dx, e.Y+dy)
}

func itemCount(e *Entity) int {
	if isBelt(e) {
		return len(e.BeltItems)
	}
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

// addBeltItem places a freshly-inserted item onto a belt, assigning it a
// stable, monotonically increasing ID so the client can track it across ticks
// and across belt-to-belt transfers.
func (w *World) addBeltItem(e *Entity, res string) {
	w.nextItemID++
	e.BeltItems = append(e.BeltItems, BeltItem{ID: w.nextItemID, Res: res, Progress: 0})
}

func beltItemCount(e *Entity, res string) int {
	count := 0
	for _, item := range e.BeltItems {
		if item.Res == res {
			count++
		}
	}
	return count
}

func removeBeltItems(e *Entity, res string, count int) {
	removed := 0
	newItems := make([]BeltItem, 0, len(e.BeltItems))
	for _, item := range e.BeltItems {
		if removed >= count {
			newItems = append(newItems, item)
			continue
		}
		if item.Res == res {
			removed++
		} else {
			newItems = append(newItems, item)
		}
	}
	e.BeltItems = newItems
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

	if isBelt(target) {
		if !cornerAcceptsFrom(from, target) {
			return false
		}
		if itemCount(target) >= MaxBeltBuffer {
			return false
		}
		if !takeItem(from, res) {
			return false
		}
		w.addBeltItem(target, res)
		w.markChanged(from)
		w.markChanged(target)
		return true
	}

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

// Credits a hub's owner with `res`, marking the player changed.
func (w *World) deliverToOwner(hub *Entity, res string) {
	p := w.players[hub.OwnerID]
	if p == nil {
		return
	}
	p.Resources[res]++
	w.changedPlayers[hub.OwnerID] = true
}

// Advances every belt: items slowly slide along the belt surface by
// progressing their position each tick. When an item reaches the end of a
// segment it is delivered to the neighboring entity, keeping the leftover
// fractional progress so movement stays continuous across belt seams.
func (w *World) tickLogistics() {
	for _, e := range w.entities {
		if !isBelt(e) || len(e.BeltItems) == 0 {
			continue
		}

		step := 1.0 / BeltSpeed
		var completed []BeltItem
		remaining := make([]BeltItem, 0, len(e.BeltItems))
		for _, item := range e.BeltItems {
			item.Progress += step
			if item.Progress >= 1.0 {
				completed = append(completed, item)
			} else {
				remaining = append(remaining, item)
			}
		}

		// Hand further-along items off first so that when several finish in
		// the same tick they stay ordered on the receiving belt.
		sort.Slice(completed, func(i, j int) bool {
			return completed[i].Progress > completed[j].Progress
		})

		e.BeltItems = remaining
		w.markChanged(e)

		if len(completed) == 0 {
			continue
		}

		target := w.neighborEntity(e)
		targetOk := target != nil
		if targetOk {
			_, targetOk = w.registry.Buildings[target.Type]
		}

		if target == nil || !targetOk {
			for i := range completed {
				completed[i].Progress = 1.0
				e.BeltItems = append(e.BeltItems, completed[i])
			}
			continue
		}

		if target.Type == "hub" {
			for _, item := range completed {
				w.deliverToOwner(target, item.Res)
			}
			w.markChanged(target)
			continue
		}

		if isBelt(target) {
			for _, item := range completed {
				if !cornerAcceptsFrom(e, target) {
					item.Progress = 1.0
					e.BeltItems = append(e.BeltItems, item)
					continue
				}
				if len(target.BeltItems) >= MaxBeltBuffer {
					item.Progress = 1.0
					e.BeltItems = append(e.BeltItems, item)
					continue
				}
				item.Progress -= 1.0
				target.BeltItems = append(target.BeltItems, item)
			}
			w.markChanged(target)
			continue
		}

		for _, item := range completed {
			addItem(target, item.Res)
		}
		w.markChanged(target)
	}
}
