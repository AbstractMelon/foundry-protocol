package world

import "foundryprotocol/protocol"

type Player struct {
	ID        string
	Name      string
	Resources map[string]int
}

func (p *Player) View() protocol.PlayerView {
	return protocol.PlayerView{
		ID:        p.ID,
		Name:      p.Name,
		Resources: p.Resources,
	}
}
