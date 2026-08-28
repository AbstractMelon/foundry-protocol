package content

import (
	"fmt"
	"sort"
)

type Registry struct {
	Resources map[string]Resource
	Buildings map[string]Building
	Recipes   map[string]Recipe
	Terrains  map[string]TerrainType
}

func New() *Registry {
	return &Registry{
		Resources: make(map[string]Resource),
		Buildings: make(map[string]Building),
		Recipes:   make(map[string]Recipe),
		Terrains:  make(map[string]TerrainType),
	}
}

func (r *Registry) AddResource(res Resource) {
	r.Resources[res.ID] = res
}

func (r *Registry) AddBuilding(b Building) {
	r.Buildings[b.ID] = b
}

func (r *Registry) AddRecipe(rec Recipe) {
	r.Recipes[rec.ID] = rec
}

func (r *Registry) AddTerrain(t TerrainType) {
	r.Terrains[t.ID] = t
}

func (r *Registry) ResourceIDs() []string {
	ids := make([]string, 0, len(r.Resources))
	for id := range r.Resources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) BuildingIDs() []string {
	ids := make([]string, 0, len(r.Buildings))
	for id := range r.Buildings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) TerrainIDs() []string {
	ids := make([]string, 0, len(r.Terrains))
	for id := range r.Terrains {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Returns every texture filename the content bundle wants, deduplicated and
// sorted for deterministic output.
func (r *Registry) ReferencedTextures() []string {
	set := make(map[string]bool)
	for _, res := range r.Resources {
		if res.Texture != "" {
			set[res.Texture] = true
		}
	}
	for _, b := range r.Buildings {
		if b.Texture != "" {
			set[b.Texture] = true
		}
	}
	for _, t := range r.Terrains {
		if t.Texture != "" {
			set[t.Texture] = true
		}
	}
	textures := make([]string, 0, len(set))
	for ref := range set {
		textures = append(textures, ref)
	}
	sort.Strings(textures)
	return textures
}

func (r *Registry) DefaultTerrain() string {
	for id, t := range r.Terrains {
		if t.Base {
			return id
		}
	}
	ids := r.TerrainIDs()
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

func (r *Registry) Validate() []string {
	var errs []string
	for id, b := range r.Buildings {
		if b.Recipe == "" {
			continue
		}
		if _, ok := r.Recipes[b.Recipe]; !ok {
			errs = append(errs, fmt.Sprintf("building %q references missing recipe %q", id, b.Recipe))
		}
	}
	for id, rec := range r.Recipes {
		for res := range rec.Input {
			if _, ok := r.Resources[res]; !ok {
				errs = append(errs, fmt.Sprintf("recipe %q references missing input resource %q", id, res))
			}
		}
		for res := range rec.Output {
			if _, ok := r.Resources[res]; !ok {
				errs = append(errs, fmt.Sprintf("recipe %q references missing output resource %q", id, res))
			}
		}
	}
	for id, res := range r.Resources {
		for _, terrain := range res.CanPlaceOn {
			if _, ok := r.Terrains[terrain]; !ok {
				errs = append(errs, fmt.Sprintf("resource %q can_place_on references missing terrain %q", id, terrain))
			}
		}
	}
	return errs
}
