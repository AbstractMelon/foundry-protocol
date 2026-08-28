package protocol

type EntityView struct {
	ID       int64          `json:"id"`
	Type     string         `json:"type"`
	OwnerID  string         `json:"owner_id"`
	X        int            `json:"x"`
	Y        int            `json:"y"`
	Health   int            `json:"health"`
	Progress int            `json:"progress"`
	Dir      int            `json:"dir"`
	Stock    map[string]int `json:"stock"`
}

type PlayerView struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Resources map[string]int `json:"resources"`
}

type TileView struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Terrain string `json:"terrain"`
	Deposit string `json:"deposit,omitempty"`
	Yield   int    `json:"yield,omitempty"`
}

type WorldSnapshot struct {
	Tick     int64        `json:"tick"`
	TileSize int          `json:"tile_size"`
	Entities []EntityView `json:"entities"`
	Players  []PlayerView `json:"players"`
	Tiles    []TileView   `json:"tiles"`
}

type Diff struct {
	Tick            int64        `json:"tick"`
	EntitiesAdded   []EntityView `json:"entities_added"`
	EntitiesChanged []EntityView `json:"entities_changed"`
	EntitiesRemoved []int64      `json:"entities_removed"`
	PlayersChanged  []PlayerView `json:"players_changed"`
	TilesChanged    []TileView   `json:"tiles_changed"`
}

type ResourceDef struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	StackSize  int      `json:"stack_size"`
	Texture    string   `json:"texture,omitempty"`
	CanPlaceOn []string `json:"can_place_on,omitempty"`
	Yield      int      `json:"yield,omitempty"`
}

type BuildingDef struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Category       string         `json:"category"`
	Color          string         `json:"color"`
	Texture        string         `json:"texture,omitempty"`
	Health         int            `json:"health"`
	Cost           map[string]int `json:"cost"`
	Recipe         string         `json:"recipe,omitempty"`
	RecipeDuration int            `json:"recipe_duration,omitempty"`
}

type TerrainDef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Color     string `json:"color"`
	Texture   string `json:"texture,omitempty"`
	Buildable bool   `json:"buildable"`
}

type ContentBundle struct {
	Resources []ResourceDef     `json:"resources"`
	Buildings []BuildingDef     `json:"buildings"`
	Terrains  []TerrainDef      `json:"terrains"`
	Textures  map[string]string `json:"textures,omitempty"`
}
