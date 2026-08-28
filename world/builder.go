package world

import (
	"foundryprotocol/protocol"
)

type ChangeSet struct {
	EntitiesAdded   []*Entity
	EntitiesChanged []*Entity
	EntitiesRemoved []int64
	PlayersChanged  []*Player
	TilesChanged    []Coord
}

func (w *World) TakeChanges() ChangeSet {
	var ch ChangeSet
	for _, id := range sortedKeyedIDs(w.addedIDs) {
		if e, ok := w.entities[id]; ok {
			ch.EntitiesAdded = append(ch.EntitiesAdded, e)
		}
	}
	for _, id := range sortedKeyedIDs(w.changedIDs) {
		if e, ok := w.entities[id]; ok {
			ch.EntitiesChanged = append(ch.EntitiesChanged, e)
		}
	}
	for _, id := range sortedKeyedInt64(w.removedIDs) {
		ch.EntitiesRemoved = append(ch.EntitiesRemoved, id)
	}
	for _, id := range sortedKeyedStrings(w.changedPlayers) {
		if p, ok := w.players[id]; ok {
			ch.PlayersChanged = append(ch.PlayersChanged, p)
		}
	}
	for _, c := range sortedTileChangeKeys(w.changedTiles) {
		ch.TilesChanged = append(ch.TilesChanged, c)
	}
	w.clearMutations()
	return ch
}

func (w *World) Snapshot() protocol.WorldSnapshot {
	entities := make([]protocol.EntityView, 0, len(w.entities))
	for _, id := range sortedEntityIDs(w.entities) {
		entities = append(entities, w.entities[id].View())
	}
	players := make([]protocol.PlayerView, 0, len(w.players))
	for _, id := range sortedPlayerIDs(w.players) {
		players = append(players, w.players[id].View())
	}
	tiles := make([]protocol.TileView, 0, len(w.tiles))
	for _, c := range sortedTileKeys(w.tiles) {
		t := w.tiles[c]
		tiles = append(tiles, protocol.TileView{X: c.X, Y: c.Y, Terrain: t.Terrain, Deposit: t.Deposit, Yield: t.Yield})
	}
	return protocol.WorldSnapshot{Tick: w.tick, TileSize: TileSize, Entities: entities, Players: players, Tiles: tiles}
}

func (w *World) BuildSnapshot() protocol.Message {
	snap := w.Snapshot()
	return protocol.Message{Type: protocol.TypeSnapshot, Snapshot: &snap}
}

func (w *World) BuildDiff(ch ChangeSet) protocol.Message {
	d := protocol.Diff{
		Tick:            w.tick,
		EntitiesAdded:   []protocol.EntityView{},
		EntitiesChanged: []protocol.EntityView{},
		EntitiesRemoved: []int64{},
		PlayersChanged:  []protocol.PlayerView{},
		TilesChanged:    []protocol.TileView{},
	}
	for _, e := range ch.EntitiesAdded {
		d.EntitiesAdded = append(d.EntitiesAdded, e.View())
	}
	for _, e := range ch.EntitiesChanged {
		d.EntitiesChanged = append(d.EntitiesChanged, e.View())
	}
	for _, id := range ch.EntitiesRemoved {
		d.EntitiesRemoved = append(d.EntitiesRemoved, id)
	}
	for _, p := range ch.PlayersChanged {
		d.PlayersChanged = append(d.PlayersChanged, p.View())
	}
	for _, c := range ch.TilesChanged {
		t := w.TerrainAt(c.X, c.Y)
		d.TilesChanged = append(d.TilesChanged, protocol.TileView{X: c.X, Y: c.Y, Terrain: t.Terrain, Deposit: t.Deposit, Yield: t.Yield})
	}
	return protocol.Message{Type: protocol.TypeDiff, Diff: &d}
}

func (w *World) BuildWelcome(playerID, playerName, worldName string, bundle protocol.ContentBundle, snap protocol.WorldSnapshot) protocol.Message {
	return protocol.Message{
		Type:       protocol.TypeWelcome,
		PlayerID:   playerID,
		PlayerName: playerName,
		Text:       worldName,
		Content:    &bundle,
		Snapshot:   &snap,
	}
}
