package world

import (
	"sort"

	"foundryprotocol/content"
)

type World struct {
	registry *content.Registry
	players  map[string]*Player
	entities map[int64]*Entity
	tick     int64
	nextID   int64
	nextItemID int64

	size         int
	tiles        map[Coord]Tile
	changedTiles map[Coord]bool

	addedIDs       map[int64]bool
	changedIDs     map[int64]bool
	removedIDs     map[int64]bool
	changedPlayers map[string]bool
}

func New(reg *content.Registry) *World {
	return &World{
		registry:       reg,
		players:        make(map[string]*Player),
		entities:       make(map[int64]*Entity),
		size:           DefaultRegionSize,
		tiles:          make(map[Coord]Tile),
		changedTiles:   make(map[Coord]bool),
		addedIDs:       make(map[int64]bool),
		changedIDs:     make(map[int64]bool),
		removedIDs:     make(map[int64]bool),
		changedPlayers: make(map[string]bool),
	}
}

func (w *World) Registry() *content.Registry {
	return w.registry
}

func (w *World) TickCount() int64 {
	return w.tick
}

func (w *World) AddPlayer(id, name string) *Player {
	p := &Player{ID: id, Name: name, Resources: map[string]int{}}
	w.players[id] = p
	w.changedPlayers[id] = true
	return p
}

func (w *World) GetPlayer(id string) *Player {
	return w.players[id]
}

func (w *World) PlayerByName(name string) *Player {
	for _, p := range w.players {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (w *World) PlayerCount() int {
	return len(w.players)
}

func (w *World) EntityAt(x, y int) *Entity {
	for _, e := range w.entities {
		if e.X == x && e.Y == y {
			return e
		}
	}
	return nil
}

func (w *World) EntityCount() int {
	return len(w.entities)
}

func (w *World) markAdded(e *Entity) {
	w.addedIDs[e.ID] = true
	delete(w.changedIDs, e.ID)
}

func (w *World) markChanged(e *Entity) {
	if w.addedIDs[e.ID] {
		return
	}
	w.changedIDs[e.ID] = true
}

func (w *World) markRemoved(e *Entity) {
	delete(w.addedIDs, e.ID)
	delete(w.changedIDs, e.ID)
	w.removedIDs[e.ID] = true
}

func (w *World) MarkPlayerChanged(id string) {
	w.changedPlayers[id] = true
}

func (w *World) RemoveEntity(e *Entity) {
	delete(w.entities, e.ID)
	w.markRemoved(e)
}

func sortedEntityIDs(entities map[int64]*Entity) []int64 {
	ids := make([]int64, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedPlayerIDs(players map[string]*Player) []string {
	ids := make([]string, 0, len(players))
	for id := range players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
