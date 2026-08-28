package world

import (
	"errors"
	"fmt"
)

func (w *World) PlaceBuilding(playerID, buildingType string, x, y int) error {
	p := w.players[playerID]
	if p == nil {
		return errors.New("unknown player")
	}
	def, ok := w.registry.Buildings[buildingType]
	if !ok {
		return fmt.Errorf("unknown building %q", buildingType)
	}
	if w.EntityAt(x, y) != nil {
		return errors.New("tile occupied")
	}
	if !w.BuildableAt(x, y) {
		return errors.New("cannot build on that terrain")
	}
	for res, qty := range def.Cost {
		if p.Resources[res] < qty {
			return fmt.Errorf("missing %d %s", qty-p.Resources[res], res)
		}
	}
	for res, qty := range def.Cost {
		p.Resources[res] -= qty
		if p.Resources[res] == 0 {
			delete(p.Resources, res)
		}
	}
	w.changedPlayers[playerID] = true

	e := &Entity{
		ID:      w.nextID,
		Type:    buildingType,
		OwnerID: playerID,
		X:       x,
		Y:       y,
		Health:  def.Health,
		Dir:     DirEast,
		Stock:   map[string]int{},
	}
	w.nextID++
	w.entities[e.ID] = e
	w.markAdded(e)
	return nil
}

func (w *World) RemoveBuilding(playerID string, entityID int64) error {
	p := w.players[playerID]
	if p == nil {
		return errors.New("unknown player")
	}
	e, ok := w.entities[entityID]
	if !ok {
		return errors.New("structure not found")
	}
	if e.OwnerID != playerID {
		return errors.New("not your structure")
	}
	if def, ok := w.registry.Buildings[e.Type]; ok {
		for res, qty := range def.Cost {
			p.Resources[res] += qty
		}
		w.changedPlayers[playerID] = true
	}
	w.RemoveEntity(e)
	return nil
}

// Adds a hub for a player at their spawn location if they do not already own
// one. The returned bool is true when a hub was created. It is idempotent so
// it can safely run on every join.
func (w *World) EnsureHub(playerID string) bool {
	for _, e := range w.entities {
		if e.Type == "hub" && e.OwnerID == playerID {
			return false
		}
	}
	x, y := w.findSpawn(playerID)
	if err := w.PlaceBuilding(playerID, "hub", x, y); err != nil {
		return false
	}
	// Hubs are account-owned amenities; give back any (zero) cost.
	return true
}

// Resolves a free, buildable tile near the origin for a player's hub.
func (w *World) findSpawn(playerID string) (int, int) {
	base := len(w.players) * 3
	for offset := 0; offset < 6; offset++ {
		x, y := 1+base, offset
		if w.EntityAt(x, y) == nil && w.BuildableAt(x, y) {
			return x, y
		}
	}
	return 1 + base, 0
}

func (w *World) ClearAllEntities() int {
	ids := sortedEntityIDs(w.entities)
	removed := 0
	for _, id := range ids {
		e := w.entities[id]
		w.RemoveEntity(e)
		removed++
	}
	if removed > 0 {
		for id := range w.players {
			w.changedPlayers[id] = true
		}
	}
	return removed
}
