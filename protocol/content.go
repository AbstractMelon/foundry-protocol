package protocol

import (
	"sort"

	"foundryprotocol/content"
)

// Converts the loaded content registry plus the data-URL assets map into the
// wire bundle. Any texture reference that is missing from the assets map is
// simply skipped; the client falls back to the definition's color, and the
// server logs a warning for it at boot.
func BuildContentBundle(reg *content.Registry, assets map[string]string) ContentBundle {
	bundle := ContentBundle{}

	resIDs := make([]string, 0, len(reg.Resources))
	for id := range reg.Resources {
		resIDs = append(resIDs, id)
	}
	sort.Strings(resIDs)
	for _, id := range resIDs {
		r := reg.Resources[id]
		bundle.Resources = append(bundle.Resources, ResourceDef{
			ID:         r.ID,
			Name:       r.Name,
			Color:      r.Color,
			StackSize:  r.StackSize,
			Texture:    r.Texture,
			CanPlaceOn: r.CanPlaceOn,
			Yield:      r.Yield,
		})
	}

	bIDs := make([]string, 0, len(reg.Buildings))
	for id := range reg.Buildings {
		bIDs = append(bIDs, id)
	}
	sort.Strings(bIDs)
	for _, id := range bIDs {
		b := reg.Buildings[id]
		def := BuildingDef{
			ID:       b.ID,
			Name:     b.Name,
			Category: b.Category,
			Color:    b.Color,
			Texture:  b.Texture,
			Health:   b.Health,
			Cost:     b.Cost,
			Recipe:   b.Recipe,
		}
		if rec, ok := reg.Recipes[b.Recipe]; ok {
			def.RecipeDuration = rec.DurationTicks
		}
		bundle.Buildings = append(bundle.Buildings, def)
	}

	tIDs := make([]string, 0, len(reg.Terrains))
	for id := range reg.Terrains {
		tIDs = append(tIDs, id)
	}
	sort.Strings(tIDs)
	for _, id := range tIDs {
		t := reg.Terrains[id]
		bundle.Terrains = append(bundle.Terrains, TerrainDef{
			ID:        t.ID,
			Name:      t.Name,
			Category:  t.Category,
			Color:     t.Color,
			Texture:   t.Texture,
			Buildable: t.Buildable,
		})
	}

	textures := make(map[string]string)
	for _, ref := range reg.ReferencedTextures() {
		if data, ok := assets[content.TextureRefKey(ref)]; ok {
			textures[ref] = data
		}
	}
	if len(textures) > 0 {
		bundle.Textures = textures
	}

	return bundle
}
