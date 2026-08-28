package server

type Config struct {
	Addr               string
	WorldName          string
	ContentDir         string
	AssetDir           string
	SaveDir            string
	TPS                int
	Dev                bool
	AutoSaveEveryTicks int64
	WorldSeed          int64
}

func (c Config) withDefaults() Config {
	if c.Addr == "" {
		c.Addr = ":8090"
	}
	if c.WorldName == "" {
		c.WorldName = "world"
	}
	if c.ContentDir == "" {
		c.ContentDir = "content"
	}
	if c.AssetDir == "" {
		c.AssetDir = "assets"
	}
	if c.SaveDir == "" {
		c.SaveDir = "saves"
	}
	if c.TPS <= 0 {
		c.TPS = 10
	}
	if c.AutoSaveEveryTicks <= 0 {
		c.AutoSaveEveryTicks = 600
	}
	return c
}
