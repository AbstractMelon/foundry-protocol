package server

import (
	"strconv"
	"strings"

	"foundryprotocol/protocol"
	"foundryprotocol/world"
)

func devSeed(w *world.World, cfg Config) {
	dev := w.AddPlayer("dev", "Dev")
	for _, id := range w.Registry().ResourceIDs() {
		dev.Resources[id] = 1000
	}
	x := 2
	for _, id := range []string{"miner", "miner", "smelter", "wall"} {
		if _, ok := w.Registry().Buildings[id]; !ok {
			continue
		}
		if id == "miner" {
			c := oreNear(w, x, 0)
			if c == nil {
				c = &world.Coord{X: x, Y: 0}
			}
			_ = w.PlaceBuilding(dev.ID, id, c.X, c.Y)
			x += 2
			continue
		}
		_ = w.PlaceBuilding(dev.ID, id, x, 0)
		x += 2
	}
	w.TakeChanges()
}

func oreNear(w *world.World, nearX, nearY int) *world.Coord {
	for radius := 0; radius <= 4; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if _, hasDeposit := w.DepositAt(nearX+dx, nearY+dy); hasDeposit {
					return &world.Coord{X: nearX + dx, Y: nearY + dy}
				}
			}
		}
	}
	return nil
}

func (s *Server) grantDevResources(playerID string) {
	p := s.world.GetPlayer(playerID)
	if p == nil {
		return
	}
	for _, id := range s.reg.ResourceIDs() {
		p.Resources[id] = 1000
	}
	s.world.MarkPlayerChanged(playerID)
}

func (s *Server) devCommand(sess *Session, text string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return
	}
	player := s.world.GetPlayer(sess.playerID)
	cmd := fields[0]
	switch cmd {
	case "/help":
		sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "info", Text: "commands: /give <resource> <qty>, /set <resource> <qty>, /clear, /regen"})
	case "/give", "/set":
		if player == nil || len(fields) < 3 {
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: "usage: /give <resource> <qty>"})
			return
		}
		qty, err := strconv.Atoi(fields[2])
		if err != nil {
			sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: "qty must be a number"})
			return
		}
		res := fields[1]
		if cmd == "/give" {
			player.Resources[res] += qty
		} else {
			player.Resources[res] = qty
		}
		s.world.MarkPlayerChanged(sess.playerID)
		sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "ok", Text: "you now have " + strconv.Itoa(player.Resources[res]) + " " + res})
	case "/clear":
		n := s.world.ClearAllEntities()
		sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "ok", Text: "removed " + strconv.Itoa(n) + " structures"})
	case "/regen":
		s.world.Generate(seedFromName(s.cfg.WorldName, s.cfg.WorldSeed))
		sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "ok", Text: "world regenerated"})
	default:
		sess.enqueue(protocol.Message{Type: protocol.TypeSystem, Value: "error", Text: "unknown command, try /help"})
	}
}

func seedFromName(name string, seed int64) int64 {
	if seed != 0 {
		return seed
	}
	return int64(uint64(hashSeed(name)) >> 1)
}
