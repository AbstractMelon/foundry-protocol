package content

type Resource struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Color      string   `yaml:"color"`
	StackSize  int      `yaml:"stack_size"`
	Texture    string   `yaml:"texture"`
	CanPlaceOn []string `yaml:"can_place_on"`
	Yield      int      `yaml:"yield"`
}

type Building struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Category string         `yaml:"category"`
	Color    string         `yaml:"color"`
	Texture  string         `yaml:"texture"`
	Health   int            `yaml:"health"`
	Cost     map[string]int `yaml:"cost"`
	Recipe   string         `yaml:"recipe"`
	Storage  map[string]int `yaml:"storage"`
}

type Recipe struct {
	ID            string         `yaml:"id"`
	Name          string         `yaml:"name"`
	Category      string         `yaml:"category"`
	Input         map[string]int `yaml:"input"`
	Output        map[string]int `yaml:"output"`
	DurationTicks int            `yaml:"duration_ticks"`
	RequiresOre   bool           `yaml:"requires_ore"`
}

type TerrainType struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Category  string `yaml:"category"`
	Color     string `yaml:"color"`
	Texture   string `yaml:"texture"`
	Buildable bool   `yaml:"buildable"`
	Base      bool   `yaml:"base"`
}
