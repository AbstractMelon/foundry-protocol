package world

import "sort"

func sortedKeyedIDs(m map[int64]bool) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func sortedKeyedInt64(m map[int64]bool) []int64 {
	return sortedKeyedIDs(m)
}

func sortedKeyedStrings(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedTileKeys(m map[Coord]Tile) []Coord {
	keys := make([]Coord, 0, len(m))
	for c := range m {
		keys = append(keys, c)
	}
	sortCoords(keys)
	return keys
}

func sortedTileChangeKeys(m map[Coord]bool) []Coord {
	keys := make([]Coord, 0, len(m))
	for c := range m {
		keys = append(keys, c)
	}
	sortCoords(keys)
	return keys
}

func sortCoords(keys []Coord) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].X != keys[j].X {
			return keys[i].X < keys[j].X
		}
		return keys[i].Y < keys[j].Y
	})
}
