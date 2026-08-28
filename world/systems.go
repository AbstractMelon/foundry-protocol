package world

import "foundryprotocol/content"

func (w *World) Tick() {
	w.tickSystems()
	w.tick++
}

func (w *World) tickSystems() {
	for _, e := range w.entities {
		b, ok := w.registry.Buildings[e.Type]
		if !ok {
			continue
		}
		if b.Category == "logistics" {
			continue
		}
		rec, ok := w.registry.Recipes[b.Recipe]
		if !ok {
			continue
		}
		if rec.Category == "extraction" {
			w.tickExtraction(e, b.Storage, rec)
			continue
		}
		w.tickProduction(e, b.Storage, rec)
	}
	w.tickLogistics()
}

func (w *World) tickProduction(e *Entity, storage map[string]int, rec content.Recipe) {
	// Pull any missing inputs from the belt feeding this machine.
	if !hasInputs(e.Stock, rec.Input) {
		w.pullInputs(e, rec.Input)
	}
	e.Progress++
	w.markChanged(e)
	if e.Progress < rec.DurationTicks {
		return
	}
	if !hasInputs(e.Stock, rec.Input) {
		e.Progress = rec.DurationTicks
		return
	}
	consume(e.Stock, rec.Input)
	produce(e.Stock, rec.Output)
	e.Progress = 0
	w.markChanged(e)
	// Push finished outputs toward the belt/hub this machine faces.
	for res := range rec.Output {
		w.receiveInto(e, res)
	}
}

// Moves items from the belt behind a machine into its stock so a smelter can
// consume what a miner ships to it.
func (w *World) pullInputs(e *Entity, rec map[string]int) {
	dx, dy := dirOffset((e.Dir + 2) % 4) // opposite of facing
	src := w.EntityAt(e.X+dx, e.Y+dy)
	if src == nil {
		return
	}
	for res, need := range rec {
		have := e.Stock[res]
		missing := need - have
		if missing <= 0 {
			continue
		}

		var srcHas int
		if isBelt(src) {
			srcHas = beltItemCount(src, res)
		} else {
			srcHas = src.Stock[res]
		}
		if srcHas <= 0 {
			continue
		}
		take := srcHas
		if take > missing {
			take = missing
		}
		e.Stock[res] += take

		if isBelt(src) {
			removeBeltItems(src, res, take)
		} else {
			src.Stock[res] -= take
			if src.Stock[res] <= 0 {
				delete(src.Stock, res)
			}
		}
		w.markChanged(e)
		w.markChanged(src)
	}
}

// Mines the resource deposit sitting on top of the tile the building occupies.
// The deposit's resource drives what is produced (ignoring the recipe output);
// the recipe only supplies the mining duration. Depleting the yield leaves the
// base terrain behind.
func (w *World) tickExtraction(e *Entity, storage map[string]int, rec content.Recipe) {
	t := w.TerrainAt(e.X, e.Y)
	if t.Deposit == "" {
		e.Progress = 0
		return
	}
	res := t.Deposit
	if cap, ok := storage[res]; ok && e.Stock[res] >= cap {
		if e.Progress != 0 {
			e.Progress = 0
			w.markChanged(e)
		}
		return
	}
	e.Progress++
	w.markChanged(e)
	if e.Progress < rec.DurationTicks {
		return
	}
	e.Stock[res]++
	w.markChanged(e)
	w.SetYield(e.X, e.Y, t.Yield-1)
	e.Progress = 0
	// Ship the freshly mined ore down the belt line if possible.
	w.receiveInto(e, res)
}

func hasInputs(stock, inputs map[string]int) bool {
	for res, qty := range inputs {
		if stock[res] < qty {
			return false
		}
	}
	return true
}

func consume(stock, inputs map[string]int) {
	for res, qty := range inputs {
		stock[res] -= qty
		if stock[res] == 0 {
			delete(stock, res)
		}
	}
}

func produce(stock, outputs map[string]int) {
	var keys []string
	for res := range outputs {
		keys = append(keys, res)
	}
	for _, res := range keys {
		stock[res] += outputs[res]
	}
}

const MaxCatchUpTicks int64 = 1_000_000

func (w *World) CatchUpProduction(ticks int64) {
	if ticks > MaxCatchUpTicks {
		ticks = MaxCatchUpTicks
	}
	for i := int64(0); i < ticks; i++ {
		w.tickSystems()
		w.tick++
	}
	w.clearMutations()
}

func (w *World) clearMutations() {
	w.addedIDs = make(map[int64]bool)
	w.changedIDs = make(map[int64]bool)
	w.removedIDs = make(map[int64]bool)
	w.changedPlayers = make(map[string]bool)
	w.changedTiles = make(map[Coord]bool)
}
