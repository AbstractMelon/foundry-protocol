package protocol

import "encoding/json"

const (
	TypeHello          = "hello"
	TypeWelcome        = "welcome"
	TypeSnapshot       = "snapshot"
	TypeDiff           = "diff"
	TypePlaceBuilding  = "place_building"
	TypeRemoveBuilding = "remove_building"
	TypeChat           = "chat"
	TypeChatReceived   = "chat_received"
	TypeSystem         = "system"
)

type Message struct {
	Type string `json:"type"`

	Name       string `json:"name,omitempty"`
	PlayerID   string `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Seq        int64  `json:"seq,omitempty"`

	BuildingType string `json:"building_type,omitempty"`
	TileX        int    `json:"tile_x,omitempty"`
	TileY        int    `json:"tile_y,omitempty"`
	Dir          int    `json:"dir,omitempty"`
	Flipped      bool   `json:"flipped,omitempty"`
	EntityID     int64  `json:"entity_id,omitempty"`
	Text         string `json:"text,omitempty"`
	Value        string `json:"value,omitempty"`

	Snapshot *WorldSnapshot `json:"snapshot,omitempty"`
	Diff     *Diff          `json:"diff,omitempty"`
	Content  *ContentBundle `json:"content,omitempty"`
}

func Decode(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	return m, err
}

func (m Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}
