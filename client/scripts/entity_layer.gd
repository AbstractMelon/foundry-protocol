extends Node2D

const FALLBACK_TILE := 48
const DEFAULT_DIR := 1
# Server simulation runs at 10 TPS with BeltSpeed=8 ticks per segment, so
# items slide 10/8 = 1.25 belt segments per real second. The client uses this
# fixed rate to dead-reckon items smoothly between server updates.
const BELT_SEGMENTS_PER_MSEC := 1.25 / 1000.0
const BELT_CORRECTION_MS := 120.0

var _tile_size := FALLBACK_TILE
var _entities := {}
var _buildings := {}
var _resources := {}
var _terrains := {}
var _terrain := {}
var _players := {}
var _textures := {}

# Per-belt render tracks so items animate continuously and smoothly between
# server updates. Keyed as _belt_tracks[entity_id][item_id] = {progress, seen}.
var _belt_tracks := {}
var _has_moving_belts := false

var _preview_def_id := ""
var _preview_tile := Vector2i.ZERO
var _preview_dir := DEFAULT_DIR
var _preview_flipped := false
var _preview_player_id := ""

var _show_grid := false


func _ready() -> void:
	texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST


func set_content(bundle: Dictionary) -> void:
	_buildings.clear()
	for b in bundle.get("buildings", []):
		_buildings[b.get("id", "")] = b
	_resources.clear()
	for r in bundle.get("resources", []):
		_resources[r.get("id", "")] = r
	_terrains.clear()
	for t in bundle.get("terrains", []):
		_terrains[t.get("id", "")] = t
	if bundle.has("tile_size"):
		_tile_size = int(bundle.get("tile_size", _tile_size))
	_parse_textures(bundle.get("textures", {}))


func apply_snapshot(snap: Dictionary) -> void:
	_tile_size = int(snap.get("tile_size", _tile_size))
	_entities.clear()
	_players.clear()
	_terrain.clear()
	_belt_tracks.clear()
	for e in snap.get("entities", []):
		_entities[int(e.get("id", -1))] = e
	for p in snap.get("players", []):
		_players[p.get("id", "")] = p
	for t in snap.get("tiles", []):
		_terrain[Vector2i(int(t.get("x", 0)), int(t.get("y", 0)))] = t
	_reconcile_belt_tracks()
	_refresh_moving_belts()
	queue_redraw()


func apply_diff(diff: Dictionary) -> void:
	for e in diff.get("entities_added", []):
		_entities[int(e.get("id", -1))] = e
	for e in diff.get("entities_changed", []):
		_entities[int(e.get("id", -1))] = e
	for eid in diff.get("entities_removed", []):
		_entities.erase(int(eid))
	for p in diff.get("players_changed", []):
		_players[p.get("id", "")] = p
	for t in diff.get("tiles_changed", []):
		_terrain[Vector2i(int(t.get("x", 0)), int(t.get("y", 0)))] = t
	_reconcile_belt_tracks()
	_refresh_moving_belts()
	queue_redraw()


# Reconciles the local belt animation tracks against the latest authoritative
# server state. Items are keyed by their stable id so a hand-off onto the next
# belt is treated as a new track rather than a mismatched index, and existing
# tracks keep animating continuously with only a gentle correction toward the
# server's position.
func _reconcile_belt_tracks() -> void:
	var now_ms := Time.get_ticks_msec()
	var live_belts := {}
	for e in _entities.values():
		if str(_buildings.get(e.get("type", ""), {}).get("category", "")) != "logistics":
			continue
		var eid := int(e.get("id", -1))
		var items: Array = e.get("belt_items", [])
		var tracks: Dictionary = _belt_tracks.get(eid, {})
		for bi in items:
			var iid := int(bi.get("id", 0))
			if iid == 0:
				continue
			live_belts[eid] = true
			var sp := clampf(float(bi.get("progress", 0.0)), 0.0, 1.0)
			if tracks.has(iid):
				var track: Dictionary = tracks[iid]
				var due_ms := float(clampi(now_ms - int(track.seen), 0, 100000))
				var current := float(track.progress) + due_ms * BELT_SEGMENTS_PER_MSEC
				# Blend toward the server's authoritative position so drift from
				# network timing corrects gently instead of snapping or stalling.
				var blend := clampf(due_ms / BELT_CORRECTION_MS, 0.0, 1.0)
				track.progress = lerpf(current, sp, blend * 0.6)
				track.seen = now_ms
			else:
				tracks[iid] = {progress = sp, seen = now_ms}
		_belt_tracks[eid] = tracks
	# Drop tracks whose item no longer exists on its belt (moved on or removed).
	for eid in _belt_tracks.keys():
		if live_belts.has(eid):
			var items: Array = _entities.get(eid, {}).get("belt_items", [])
			var live := {}
			for bi in items:
				live[int(bi.get("id", 0))] = true
			var tracks: Dictionary = _belt_tracks[eid]
			for iid in tracks.keys():
				if not live.has(int(iid)):
					tracks.erase(iid)
		else:
			_belt_tracks.erase(eid)


# Rendered position of a track at the given time, dead-reckoned forward from
# its last known (progress, seen) at the constant belt speed, clamped to the
# segment so backed-up items stop at the belt exit.
func _render_progress(track: Dictionary, now_ms: int) -> float:
	var elapsed := float(clampi(now_ms - int(track.seen), 0, 100000))
	return clampf(float(track.progress) + elapsed * BELT_SEGMENTS_PER_MSEC, 0.0, 1.0)


func _process(_delta: float) -> void:
	if not _has_moving_belts:
		return
	# Belts animate continuously via local dead reckoning, so we re-draw every
	# frame while any cargo is in flight instead of only between server updates.
	queue_redraw()


func _refresh_moving_belts() -> void:
	_has_moving_belts = false
	for e in _entities.values():
		var def: Dictionary = _buildings.get(e.get("type", ""), {})
		var items: Array = e.get("belt_items", [])
		if str(def.get("category", "")) == "logistics" and not items.is_empty():
			_has_moving_belts = true
			return


func world_to_tile(v: Vector2) -> Vector2i:
	return Vector2i(floori(v.x / float(_tile_size)), floori(v.y / float(_tile_size)))


func tile_center(x: int, y: int) -> Vector2:
	return Vector2(float(x) * _tile_size + _tile_size * 0.5, float(y) * _tile_size + _tile_size * 0.5)


func tile_rect(x: int, y: int) -> Rect2:
	return Rect2(Vector2(float(x) * _tile_size, float(y) * _tile_size), Vector2(_tile_size, _tile_size))


func terrain_at(t: Vector2i) -> String:
	var tile: Dictionary = _terrain.get(t, {})
	return str(tile.get("terrain", "grass"))


func buildable_at(t: Vector2i) -> bool:
	var def: Dictionary = _terrains.get(terrain_at(t), {})
	return bool(def.get("buildable", true))


func set_preview(def_id: String, tile: Vector2i, dir: int, player_id: String, flipped := false) -> void:
	if def_id == _preview_def_id and tile == _preview_tile and dir == _preview_dir \
			and player_id == _preview_player_id and flipped == _preview_flipped:
		return
	_preview_def_id = def_id
	_preview_tile = tile
	_preview_dir = dir
	_preview_flipped = flipped
	_preview_player_id = player_id
	queue_redraw()


func clear_preview() -> void:
	if _preview_def_id == "":
		return
	_preview_def_id = ""
	queue_redraw()


func entity_at_tile(t: Vector2i) -> Dictionary:
	for e in _entities.values():
		if e.get("x", 0) == t.x and e.get("y", 0) == t.y:
			return e
	return {}


func entity_count() -> int:
	return _entities.size()


func player_name(id: String) -> String:
	var p: Dictionary = _players.get(id, {})
	return str(p.get("name", id))


func get_player(id: String) -> Dictionary:
	return _players.get(id, {})


func resource_def(id: String) -> Dictionary:
	return _resources.get(id, {})


func _parse_textures(textures: Dictionary) -> void:
	_textures.clear()
	for name in textures:
		var tex := _texture_from_data_url(str(textures[name]))
		if tex != null:
			_textures[str(name)] = tex


func set_show_grid(on: bool) -> void:
	_show_grid = on
	queue_redraw()


func _texture_from_data_url(data_url: String) -> Texture2D:
	var idx := data_url.find(",")
	if idx < 0:
		return null
	var raw := Marshalls.base64_to_raw(data_url.substr(idx + 1))
	if raw.is_empty():
		return null
	var img := Image.new()
	var err := ERR_FILE_UNRECOGNIZED
	if data_url.begins_with("data:image/jpeg") or data_url.begins_with("data:image/jpg"):
		err = img.load_jpg_from_buffer(raw)
	elif data_url.begins_with("data:image/webp"):
		err = img.load_webp_from_buffer(raw)
	else:
		err = img.load_png_from_buffer(raw)
	if err != OK:
		return null
	return ImageTexture.create_from_image(img)


func _texture_for(def: Dictionary) -> Texture2D:
	var name := str(def.get("texture", ""))
	if name == "" or not _textures.has(name):
		return null
	return _textures[name]


func _visible_tile_bounds() -> Rect2i:
	var canvas := get_canvas_transform()
	var view := get_viewport_rect()
	var top_left: Vector2 = canvas.affine_inverse() * view.position
	var bottom_right: Vector2 = canvas.affine_inverse() * (view.position + view.size)
	var t0 := world_to_tile(top_left) - Vector2i.ONE
	var t1 := world_to_tile(bottom_right) + Vector2i.ONE
	return Rect2i(t0, t1 - t0)


func _draw() -> void:
	_draw_terrain()
	_draw_entities()
	_draw_preview()


# Base terrain is drawn first, then any resource deposit is layered on top so
# ore reads as a transparent overlay on whatever block sits beneath it.
func _draw_terrain() -> void:
	var bounds := _visible_tile_bounds()
	var grid_color := Color(1, 1, 1, 0.06)
	for x in range(bounds.position.x, bounds.position.x + bounds.size.x + 1):
		for y in range(bounds.position.y, bounds.position.y + bounds.size.y + 1):
			var t := Vector2i(x, y)
			var tile: Dictionary = _terrain.get(t, {})
			var id := str(tile.get("terrain", "grass"))
			var def: Dictionary = _terrains.get(id, {})
			var color := Color.from_string(str(def.get("color", "#4c8c3f")), Color(0.3, 0.55, 0.25))
			var rect := tile_rect(x, y)
			var tex := _texture_for(def)
			if tex != null:
				draw_texture_rect(tex, rect, false)
			else:
				draw_rect(rect, color)
			if _show_grid:
				draw_line(rect.position, rect.position + Vector2(rect.size.x, 0), grid_color, 1.0)
				draw_line(rect.position, rect.position + Vector2(0, rect.size.y), grid_color, 1.0)

			var dep := str(tile.get("deposit", ""))
			if dep != "":
				var rdef: Dictionary = _resources.get(dep, {})
				var rtex := _texture_for(rdef)
				if rtex != null:
					draw_texture_rect(rtex, rect, false)
				else:
					var dep_col := Color.from_string(str(rdef.get("color", "#e8835b")), Color(0.9, 0.5, 0.35))
					draw_circle(rect.get_center(), _tile_size * 0.2, dep_col)
				var yield_left: int = int(tile.get("yield", 0))
				var max_yield := maxi(int(rdef.get("yield", 1)), 1)
				var frac := clampf(float(yield_left) / float(max_yield), 0.0, 1.0)
				draw_rect(Rect2(rect.position, Vector2(_tile_size * frac, 2.0)), Color(1, 0.9, 0.4, 0.8))


func _draw_entities() -> void:
	var font := ThemeDB.fallback_font
	# Pass 1: every entity body (including belt tiles) so no single tile can
	# overdraw a neighbour. Body drawing happens before any cargo so items
	# always sit on top and never slip behind the next belt tile at a seam.
	for e in _entities.values():
		var def: Dictionary = _buildings.get(e.get("type", ""), {})
		var pos := tile_center(e.get("x", 0), e.get("y", 0))
		var dir := int(e.get("dir", DEFAULT_DIR))
		var flipped := bool(e.get("flipped", false))
		_draw_entity_body(def, pos, dir, Color.WHITE, flipped)

	# Pass 2: belt cargo on top of all belt/building bodies, so an item crossing
	# onto the next belt stays fully visible across the seam.
	for e in _entities.values():
		var def: Dictionary = _buildings.get(e.get("type", ""), {})
		var pos := tile_center(e.get("x", 0), e.get("y", 0))
		var dir := int(e.get("dir", DEFAULT_DIR))
		var flipped := bool(e.get("flipped", false))
		var category: String = str(def.get("category", ""))
		if category == "logistics":
			_draw_logistics(e, pos)
			continue

		var max_h: float = def.get("health", 100)
		if max_h > 0.0:
			var hp := clampf(float(e.get("health", 0)) / max_h, 0.0, 1.0)
			var bar_y := pos.y - _tile_size * 0.5 - 5.0
			draw_rect(Rect2(pos.x - _tile_size * 0.5, bar_y, _tile_size, 3.0), Color(0.1, 0.1, 0.1))
			draw_rect(Rect2(pos.x - _tile_size * 0.5, bar_y, _tile_size * hp, 3.0), Color(0.25, 0.85, 0.35))

		var dur: int = def.get("recipe_duration", 0)
		if dur > 0:
			var frac := clampf(float(e.get("progress", 0)) / float(dur), 0.0, 1.0)
			var bar_y := pos.y + _tile_size * 0.5 + 2.0
			draw_rect(Rect2(pos.x - _tile_size * 0.5, bar_y, _tile_size, 3.0), Color(0.1, 0.1, 0.1))
			draw_rect(Rect2(pos.x - _tile_size * 0.5, bar_y, _tile_size * frac, 3.0), Color(0.95, 0.7, 0.2))

		draw_string(font, pos + Vector2(-_tile_size * 0.5 + 3, -_tile_size * 0.5 - 8),
				str(e.get("type", "")), HORIZONTAL_ALIGNMENT_LEFT, -1, 10, Color.WHITE)


func _draw_entity_body(def: Dictionary, pos: Vector2, dir: int, modulate := Color.WHITE, flipped := false) -> void:
	var color := Color.from_string(def.get("color", "#888888"), Color(0.55, 0.55, 0.55))
	var rect := Rect2(pos - Vector2(_tile_size * 0.5, _tile_size * 0.5), Vector2(_tile_size, _tile_size))
	var tex := _texture_for(def)
	if tex != null:
		if str(def.get("id", "")) == "belt_turn":
			draw_set_transform(pos, _corner_angle(dir), Vector2(-1.0 if flipped else 1.0, 1.0))
		else:
			draw_set_transform(pos, _dir_angle(dir), Vector2.ONE)
		draw_texture_rect(tex, Rect2(-Vector2(_tile_size * 0.5, _tile_size * 0.5), Vector2(_tile_size, _tile_size)), false, modulate)
		draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
	else:
		draw_rect(rect, color * modulate)


func _dir_angle(dir: int) -> float:
	return float((dir - DEFAULT_DIR) % 4) * PI * 0.5


# A corner belt sprite's base orientation turns from the West (input) side to
# the South (output) side, so rotating it to face an arbitrary output direction
# uses South (2) as the zero-offset rather than East.
func _corner_angle(dir: int) -> float:
	return float(dir - 2) * PI * 0.5


# The absolute direction from which a corner belt accepts items. Flip mirrors
# the bend around the vertical, swapping the input side.
func _corner_input_dir(dir: int, flipped: bool) -> int:
	return (dir + (3 if flipped else 1)) % 4


func _side_center(pos: Vector2, dir: int) -> Vector2:
	var half := _tile_size * 0.5
	match dir:
		0:
			return pos + Vector2(0.0, -half)
		1:
			return pos + Vector2(half, 0.0)
		2:
			return pos + Vector2(0.0, half)
		_:
			return pos + Vector2(-half, 0.0)


# Position of an item (progress 0..1) travelling along a corner's L-shaped
# path: in from the input edge, around the bend at the tile centre, out the
# output edge.
func _corner_item_pos(pos: Vector2, dir: int, flipped: bool, progress: float) -> Vector2:
	var center := pos
	var inp := _corner_input_dir(dir, flipped)
	if progress < 0.5:
		return _side_center(pos, inp).lerp(center, progress / 0.5)
	return center.lerp(_side_center(pos, dir), (progress - 0.5) / 0.5)


func _draw_preview() -> void:
	if _preview_def_id == "":
		return
	var def: Dictionary = _buildings.get(_preview_def_id, {})
	if def.is_empty():
		return
	var rect := tile_rect(_preview_tile.x, _preview_tile.y)
	var valid := _preview_valid(def)
	var tint := Color(0.3, 0.9, 0.4, 0.35) if valid else Color(0.95, 0.35, 0.3, 0.4)
	draw_rect(rect, tint)
	draw_rect(rect.grow(-2.0), Color(0.6, 1.0, 0.7) if valid else Color(0.98, 0.4, 0.35), false, 1.5)
	var pos := tile_center(_preview_tile.x, _preview_tile.y)
	var ghost := Color(0.75, 1.0, 0.8, 0.5) if valid else Color(1.0, 0.6, 0.55, 0.5)
	_draw_entity_body(def, pos, _preview_dir, ghost, _preview_flipped)
	if str(def.get("category", "")) == "logistics":
		_draw_logistics({"type": _preview_def_id, "dir": _preview_dir, "flipped": _preview_flipped}, pos)


func _preview_valid(def: Dictionary) -> bool:
	if not entity_at_tile(_preview_tile).is_empty():
		return false
	if not buildable_at(_preview_tile):
		return false
	var player: Dictionary = _players.get(_preview_player_id, {})
	var mats: Dictionary = player.get("resources", {})
	for res in def.get("cost", {}):
		if int(mats.get(res, 0)) < int(def["cost"][res]):
			return false
	return true


func _draw_logistics(e: Dictionary, pos: Vector2) -> void:
	var belt_items: Array = e.get("belt_items", [])
	if belt_items.is_empty():
		return

	var dir: int = int(e.get("dir", DEFAULT_DIR))
	var flipped := bool(e.get("flipped", false))
	var is_corner := str(e.get("type", "")) == "belt_turn"
	var dir_vec := _dir_vector(dir)
	var half := _tile_size * 0.25
	var now_ms := Time.get_ticks_msec()
	var tracks: Dictionary = _belt_tracks.get(int(e.get("id", -1)), {})

	for bi in belt_items:
		var iid := int(bi.get("id", 0))
		var res := str(bi.get("res", ""))
		if iid == 0:
			# Belt items without an id (legacy/snapshot before ids) fall back to
			# a static render so we never draw nothing.
			_draw_belt_item(res, pos, is_corner, flipped, dir, dir_vec, half, float(bi.get("progress", 0.0)))
			continue
		var track: Dictionary = tracks.get(iid, {})
		if track.is_empty():
			continue
		var progress := _render_progress(track, now_ms)
		_draw_belt_item(res, pos, is_corner, flipped, dir, dir_vec, half, progress)


func _draw_belt_item(res: String, pos: Vector2, is_corner: bool, flipped: bool, dir: int, dir_vec: Vector2, half: float, progress: float) -> void:
	var rdef: Dictionary = _resources.get(res, {})
	var col := Color.from_string(str(rdef.get("color", "#888")), Color(0.6, 0.6, 0.6))
	var item_pos: Vector2
	if is_corner:
		item_pos = _corner_item_pos(pos, dir, flipped, clampf(progress, 0.0, 1.0))
	else:
		item_pos = pos + dir_vec * (clampf(progress, 0.0, 1.0) - 0.5) * float(_tile_size)
	var tex := _texture_for(rdef)
	if tex != null:
		draw_texture_rect(tex, Rect2(item_pos - Vector2(half, half), Vector2(half * 2.0, half * 2.0)), false)
	else:
		draw_circle(item_pos, _tile_size * 0.16, col)


func _dir_vector(dir: int) -> Vector2:
	match dir:
		0:
			return Vector2.UP
		1:
			return Vector2.RIGHT
		2:
			return Vector2.DOWN
		_:
			return Vector2.LEFT
