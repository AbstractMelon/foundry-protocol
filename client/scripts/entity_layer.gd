extends Node2D

const FALLBACK_TILE := 48

var _tile_size := FALLBACK_TILE
var _entities := {}
var _buildings := {}
var _resources := {}
var _terrains := {}
var _terrain := {}
var _players := {}
var _textures := {}


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
	for e in snap.get("entities", []):
		_entities[int(e.get("id", -1))] = e
	for p in snap.get("players", []):
		_players[p.get("id", "")] = p
	for t in snap.get("tiles", []):
		_terrain[Vector2i(int(t.get("x", 0)), int(t.get("y", 0)))] = t
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
	queue_redraw()


func world_to_tile(v: Vector2) -> Vector2i:
	return Vector2i(floori(v.x / float(_tile_size)), floori(v.y / float(_tile_size)))


func tile_center(x: int, y: int) -> Vector2:
	return Vector2(float(x) * _tile_size + _tile_size * 0.5, float(y) * _tile_size + _tile_size * 0.5)


func tile_rect(x: int, y: int) -> Rect2:
	return Rect2(Vector2(float(x) * _tile_size, float(y) * _tile_size), Vector2(_tile_size, _tile_size))


func terrain_at(t: Vector2i) -> String:
	var tile: Dictionary = _terrain.get(t, {})
	return str(tile.get("terrain", "grass"))


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
	for e in _entities.values():
		var def: Dictionary = _buildings.get(e.get("type", ""), {})
		var pos := tile_center(e.get("x", 0), e.get("y", 0))
		var color := Color.from_string(def.get("color", "#888888"), Color(0.55, 0.55, 0.55))
		var rect := Rect2(pos - Vector2(_tile_size * 0.5, _tile_size * 0.5), Vector2(_tile_size, _tile_size))
		var tex := _texture_for(def)
		if tex != null:
			draw_texture_rect(tex, rect, false)
			draw_rect(rect.grow(-2.0), Color(0, 0, 0, 0.18), false, 1.0)
		else:
			draw_rect(rect, color)
			draw_rect(rect.grow(-2.0), Color(0, 0, 0, 0.35), false, 1.0)

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


func _draw_logistics(e: Dictionary, pos: Vector2) -> void:
	var def: Dictionary = _buildings.get(e.get("type", ""), {})
	var type_id: String = str(e.get("type", ""))
	var r := _tile_size * 0.5
	var inner := 0.34
	if type_id == "hub":
		draw_circle(pos, r * inner, Color(0.05, 0.25, 0.4, 0.9))
		draw_circle(pos, r * inner, Color.WHITE, false, 1.5)
		draw_string(ThemeDB.fallback_font, pos + Vector2(-4, 5), "H", HORIZONTAL_ALIGNMENT_LEFT, -1, 12, Color.WHITE)
		return

	var dir: int = int(e.get("dir", 1))
	var dir_vec := _dir_vector(dir)
	var arrow_w := r * 0.5
	var arrow_l := r * 0.45
	var base := pos
	var tip := pos + dir_vec * arrow_l
	var normal := Vector2(-dir_vec.y, dir_vec.x)
	draw_line(base, tip, Color(0.9, 0.95, 1.0, 0.9), 2.0)
	draw_line(tip, tip - (dir_vec + normal) * arrow_w * 0.5, Color(0.9, 0.95, 1.0, 0.9), 2.0)
	draw_line(tip, tip - (dir_vec - normal) * arrow_w * 0.5, Color(0.9, 0.95, 1.0, 0.9), 2.0)

	var stock: Dictionary = e.get("stock", {})
	var idx := 0
	for res in stock:
		var res_def: Dictionary = _resources.get(res, {})
		var col := Color.from_string(str(res_def.get("color", "#888")), Color(0.6, 0.6, 0.6))
		for _i in range(int(stock[res])):
			var angle := -PI * 0.5 + float(idx) * 0.7
			var dot_pos := pos + Vector2(cos(angle), sin(angle)) * r * 0.28
			draw_circle(dot_pos, 3.0, col)
			idx += 1


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