package world

import (
	"encoding/binary"
	"hash/fnv"
	"math"

	"foundryprotocol/content"
)

const DefaultRegionSize = 40

// Noise field tuning. The base scale controls how big the largest features
// are expressed in tiles: a tile coordinate multiplied by the scale feeds
// into noise whose lattice spacing is one. So (~1/scale) tiles per feature.
const (
	elevScale  = 0.035 // rolling terrain, continent-sized humps
	moistScale = 0.05  // moisture patches
	oreScale   = 0.14  // ore vein fields

	waterLevel = -0.4 // normalized elevation below this becomes water
	beachLevel = 0.0  // low, wet land right above water becomes sand
	rockLevel  = 1.0  // high ground becomes rocky highlands

	desertLevel = -0.18 // very dry land becomes desert sand
	desertSlope = 0.12  // deserts only form away from the coast
	dryLevel    = -0.03 // drier than this, grass turns dry

	oreThreshold = 0.60 // fbm field above this carries a vein
	oreDistrict  = 8    // tile size of one ore-type district
	spawnRadius  = 5    // cleared grass meadow around the origin
)

const noiseOctaveStride int64 = 2654435761

func (w *World) Generate(seed int64) {
	w.tiles = make(map[Coord]Tile)
	w.changedTiles = make(map[Coord]bool)

	side := (2*w.size + 1) * (2*w.size + 1)
	elev := make(map[Coord]float64, side)
	moist := make(map[Coord]float64, side)

	raw := make(map[Coord]float64, side)
	for x := -w.size; x <= w.size; x++ {
		for y := -w.size; y <= w.size; y++ {
			c := Coord{X: x, Y: y}
			raw[c] = 2*fbmAt(float64(x)*elevScale, float64(y)*elevScale, seed, 4) - 1
			moist[c] = 2*fbmAt(float64(x)*moistScale, float64(y)*moistScale, seed+1, 3) - 1
		}
	}

	// Standardize the field so every seed gets the same land/water balance,
	// then add a gentle dome at the origin so the spawn area stays high.
	mean, std := fieldStats(raw)
	for c := range raw {
		e := (raw[c]-mean)/std + spawnBias(c.X, c.Y, w.size)/std
		elev[c] = clamp(e, -1, 1)
	}

	for c, e := range elev {
		w.SetTerrain(c.X, c.Y, w.biomeFor(e, moist[c]), 0)
	}

	w.placeOres(seed, elev)
	w.stampSpawn()
	w.placeVoidBorder()

	for c := range w.tiles {
		w.changedTiles[c] = true
	}
}

// Maps an elevation/moisture pair to a terrain id. Ordering matters:
// coastline bands are decided first, then only dry ground gets dry variants.
func (w *World) biomeFor(e, m float64) string {
	switch {
	case e < waterLevel:
		return "water"
	case e < beachLevel:
		return "sand"
	case e >= rockLevel:
		return "rock"
	case m < desertLevel && e >= desertSlope:
		return "sand"
	case m < dryLevel:
		return "grass_dry"
	default:
		return "grass"
	}
}

// Grows coherent deposit blobs where the ore field rises above a
// threshold, and keeps a single ore type within each district so veins read
// as one material rather than a scatter of unrelated tiles. Each deposit sits
// on top of its natural base terrain (grass/rock/sand) and only spawns where
// the resource's can_place_on list allows it.
func (w *World) placeOres(seed int64, elev map[Coord]float64) {
	deposits := w.depositResources()
	if len(deposits) == 0 {
		return
	}
	for c := range elev {
		if w.TerrainAt(c.X, c.Y).Terrain == "water" {
			continue
		}
		region := fbmAt(float64(c.X)*oreScale, float64(c.Y)*oreScale, seed+2, 3)
		if region < oreThreshold {
			continue
		}
		res := w.pickOreDeposit(deposits, latticeField(c.X/oreDistrict, c.Y/oreDistrict, seed+3))
		base := w.TerrainAt(c.X, c.Y).Terrain
		if !containsStr(res.CanPlaceOn, base) {
			continue
		}
		w.SetDeposit(c.X, c.Y, base, res.ID, res.Yield)
	}
}

// Returns the resources that can spawn in the world as mine deposits
// Matches those with a yield and a list of terrain they can sit on.
func (w *World) depositResources() []content.Resource {
	var out []content.Resource
	for _, id := range w.registry.ResourceIDs() {
		res := w.registry.Resources[id]
		if res.Yield > 0 && len(res.CanPlaceOn) > 0 {
			out = append(out, res)
		}
	}
	return out
}

func (w *World) pickOreDeposit(deposits []content.Resource, kind float64) content.Resource {
	idx := int(kind * float64(len(deposits)))
	if idx >= len(deposits) {
		idx = len(deposits) - 1
	}
	return deposits[idx]
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Carves a small buildable meadow at the origin so players always have
// coherent dry ground to start on. Ore veins already placed are kept.
func (w *World) stampSpawn() {
	for dx := -spawnRadius; dx <= spawnRadius; dx++ {
		for dy := -spawnRadius; dy <= spawnRadius; dy++ {
			if dx*dx+dy*dy > spawnRadius*spawnRadius {
				continue
			}
			c := Coord{X: dx, Y: dy}
			if !w.InBounds(c.X, c.Y) {
				continue
			}
			if w.TerrainAt(c.X, c.Y).Deposit != "" {
				continue
			}
			w.SetTerrain(c.X, c.Y, "grass", 0)
		}
	}
}

func (w *World) placeVoidBorder() {
	for x := -w.size - 2; x <= w.size+2; x++ {
		for y := -w.size - 2; y <= w.size+2; y++ {
			if w.InBounds(x, y) {
				continue
			}
			w.SetTerrain(x, y, "void", 0)
		}
	}
	// ring the playable boundary with water so the edge reads visually.
	for x := -w.size; x <= w.size; x++ {
		w.SetTerrain(x, -w.size, "water", 0)
		w.SetTerrain(x, w.size, "water", 0)
		w.SetTerrain(-w.size, x, "water", 0)
		w.SetTerrain(w.size, x, "water", 0)
	}
}

// Adds a gentle parabolic dome centred on the origin, peaking at
// ~0.35 and falling to zero at the map edge, so the spawn meadow sits above
// the water line. Expressed in raw order-of-magnitude units.
func spawnBias(x, y, size int) float64 {
	frac := math.Hypot(float64(x), float64(y)) / float64(size)
	if frac >= 1 {
		return 0
	}
	d := 1 - frac
	return 0.35 * d * d
}

// Returns the mean and standard deviation of a noise field, used to normalize
// it so maps are consistent no matter where the noise happens to sit.
func fieldStats(field map[Coord]float64) (mean, std float64) {
	sum := 0.0
	for _, v := range field {
		sum += v
	}
	mean = sum / float64(len(field))
	for _, v := range field {
		d := v - mean
		std += d * d
	}
	std = math.Sqrt(std / float64(len(field)))
	return mean, std
}

// Sums octaves of smooth noise (fractal Brownian motion) to build a
// single coherent field with both large-scale shape and fine detail.
func fbmAt(x, y float64, seed int64, octaves int) float64 {
	amp := 1.0
	freq := 1.0
	sum := 0.0
	norm := 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * smoothNoise(x*freq, y*freq, seed+int64(i)*noiseOctaveStride)
		norm += amp
		amp *= 0.55
		freq *= 2.1
	}
	return sum / norm
}

// Bilinearly interpolated value noise. Lattice points hold a stable hash;
// positions between them fade smoothly, so nearby tiles are coherent rather
// than independent random pixels.
func smoothNoise(x, y float64, seed int64) float64 {
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	fx := x - x0
	fy := y - y0
	a := latticeField(int(x0), int(y0), seed)
	b := latticeField(int(x0)+1, int(y0), seed)
	c := latticeField(int(x0), int(y0)+1, seed)
	d := latticeField(int(x0)+1, int(y0)+1, seed)
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)
	return lerp(lerp(a, b, ux), lerp(c, d, ux), uy)
}

// Hashes a lattice point (plus seed) into a stable [0, 1) value.
func latticeField(x, y int, seed int64) float64 {
	h := fnv.New64a()
	var b [24]byte
	binary.LittleEndian.PutUint64(b[0:], uint64(seed))
	binary.LittleEndian.PutUint64(b[8:], uint64(x))
	binary.LittleEndian.PutUint64(b[16:], uint64(y))
	_, _ = h.Write(b[:])
	return float64(h.Sum64()>>11) / float64(1<<53)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
