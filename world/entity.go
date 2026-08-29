package world

import "foundryprotocol/protocol"

type BeltItem struct {
	ID       int64
	Res      string
	Progress float64
}

type Entity struct {
	ID        int64
	Type      string
	OwnerID   string
	X         int
	Y         int
	Health    int
	Progress  int
	Dir       int
	Flipped   bool
	Stock     map[string]int
	BeltItems []BeltItem
}

func (e *Entity) View() protocol.EntityView {
	beltItems := make([]protocol.BeltItem, len(e.BeltItems))
	for i, bi := range e.BeltItems {
		beltItems[i] = protocol.BeltItem{ID: bi.ID, Res: bi.Res, Progress: bi.Progress}
	}
	return protocol.EntityView{
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
	}
}
