package world

import "foundryprotocol/protocol"

type Entity struct {
	ID       int64
	Type     string
	OwnerID  string
	X        int
	Y        int
	Health   int
	Progress int
	Dir      int
	Stock    map[string]int
}

func (e *Entity) View() protocol.EntityView {
	return protocol.EntityView{
		ID:       e.ID,
		Type:     e.Type,
		OwnerID:  e.OwnerID,
		X:        e.X,
		Y:        e.Y,
		Health:   e.Health,
		Progress: e.Progress,
		Dir:      e.Dir,
		Stock:    e.Stock,
	}
}
