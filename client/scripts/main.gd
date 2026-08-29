extends Node2D

@onready var layer: Node2D = $EntityLayer
@onready var camera: Camera2D = $Camera2D
@onready var hud_label: Label = $UI/Root/HUD
@onready var materials_label: Label = $UI/Root/Materials
@onready var chat_log: Label = $UI/Root/ChatLog
@onready var console: LineEdit = $UI/Root/Console
@onready var console_hint: Label = $UI/Root/ConsoleHint
@onready var palette: HBoxContainer = $UI/Root/Palette

var _buildings: Dictionary = {}
var _selected := ""
var _remove_mode := false
var _rotation := 1
var _chat_lines: Array[String] = []
var _frame := 0
var _panning := false
var _pan_origin := Vector2.ZERO
var _cam_origin := Vector2.ZERO
var _debug_visible := false
var _show_grid := false

var _drag_active := false
var _drag_start_tile := Vector2i.ZERO
var _drag_started_on_belt := false
var _drag_seed_dir := 1
var _drag_seed_id := -1
var _drag_axis_locked := false
var _drag_axis := 0
var _drag_placed := {}

const PAN_SPEED := 420.0
const PAN_SPEED_FAST := 1250.0
const ZOOM_STEP := 1.12
const ZOOM_MIN := 0.35
const ZOOM_MAX := 3.0


func _ready() -> void:
	palette.visible = false
	console.visible = false
	console_hint.visible = true

	NetworkManager.welcome_received.connect(_on_welcome)
	NetworkManager.snapshot_received.connect(_on_snapshot)
	NetworkManager.diff_received.connect(_on_diff)
	NetworkManager.chat_received.connect(_on_chat_received)
	NetworkManager.system_received.connect(_on_system_received)
	NetworkManager.disconnected.connect(_on_disconnected)
	console.text_submitted.connect(_on_console_submitted)

	if NetworkManager.auto_connect:
		hud_label.text = "connecting to %s..." % NetworkManager.ws_url
	else:
		NetworkManager.connect_to_server(NetworkManager.ws_url)
		hud_label.text = "connecting to %s..." % NetworkManager.ws_url

	_set_debug(false)


func _process(delta: float) -> void:
	if not console.has_focus():
		var dir := Vector2.ZERO
		if Input.is_key_pressed(KEY_W) or Input.is_key_pressed(KEY_UP):
			dir.y -= 1.0
		if Input.is_key_pressed(KEY_S) or Input.is_key_pressed(KEY_DOWN):
			dir.y += 1.0
		if Input.is_key_pressed(KEY_A) or Input.is_key_pressed(KEY_LEFT):
			dir.x -= 1.0
		if Input.is_key_pressed(KEY_D) or Input.is_key_pressed(KEY_RIGHT):
			dir.x += 1.0
		if dir != Vector2.ZERO:
			var speed := PAN_SPEED
			if Input.is_key_pressed(KEY_SHIFT):
				speed = PAN_SPEED_FAST
			camera.position += dir.normalized() * speed * delta

	_frame += 1
	_update_belt_drag()
	_update_preview()
	if _frame % 15 != 0:
		return
	materials_label.text = _materials_text(layer.get_player(NetworkManager.player_id))
	if not _debug_visible:
		return
	var state := "offline"
	if NetworkManager.is_connected_to_server():
		state = "online"
	var ping := NetworkManager.get_ping()
	hud_label.text = "state: %s  |  fps: %d  |  tick: %d  |  entities: %d  |  ping: %dms" % [
		state, Engine.get_frames_per_second(), NetworkManager.last_tick, layer.entity_count(), int(ping)
	]


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_MIDDLE:
		if event.pressed:
			_panning = true
			_pan_origin = event.position
			_cam_origin = camera.position
		else:
			_panning = false
		return
	if event is InputEventMouseButton and event.pressed:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			_zoom(ZOOM_STEP)
			return
		if event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			_zoom(1.0 / ZOOM_STEP)
			return
	if event is InputEventMouseMotion and _panning:
		camera.position = _cam_origin - (event.position - _pan_origin)
		return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and not event.pressed:
		if _drag_active:
			_end_belt_drag()
		return
	if event is InputEventMouseButton and event.pressed and NetworkManager.is_connected_to_server():
		var tile: Vector2i = layer.world_to_tile(get_global_mouse_position())
		if event.button_index == MOUSE_BUTTON_RIGHT:
			var e: Dictionary = layer.entity_at_tile(tile)
			if not e.is_empty():
				NetworkManager.remove(int(e.get("id", -1)))
		elif event.button_index == MOUSE_BUTTON_LEFT:
			if _remove_mode:
				var e: Dictionary = layer.entity_at_tile(tile)
				if not e.is_empty():
					NetworkManager.remove(int(e.get("id", -1)))
			elif _selected == "belt":
				_start_belt_drag(tile)
			elif _selected != "":
				NetworkManager.place(_selected, tile.x, tile.y, _rotation)


# Begins a belt drag from the tile under the cursor. If a belt already occupies
# that tile its output direction is remembered so a perpendicular drag auto-adds
# a corner to route around the bend.
func _start_belt_drag(tile: Vector2i) -> void:
	_drag_active = true
	_drag_start_tile = tile
	_drag_axis_locked = false
	_drag_axis = 0
	_drag_placed.clear()
	var e: Dictionary = layer.entity_at_tile(tile)
	if not e.is_empty() and (str(e.get("type", "")) == "belt" or str(e.get("type", "")) == "belt_turn"):
		_drag_started_on_belt = true
		_drag_seed_dir = int(e.get("dir", 1))
		_drag_seed_id = int(e.get("id", -1))
	else:
		_drag_started_on_belt = false
		_drag_seed_dir = 1
		_drag_seed_id = -1


# Ends the active belt drag. A press without any movement is a plain click, so
# it still drops a single belt (unless it landed on an existing belt).
func _end_belt_drag() -> void:
	if not _drag_active:
		return
	_drag_active = false
	if not _drag_axis_locked and not _drag_started_on_belt:
		NetworkManager.place("belt", _drag_start_tile.x, _drag_start_tile.y, _rotation)
	_drag_axis_locked = false
	_drag_started_on_belt = false
	_drag_seed_dir = 1
	_drag_seed_id = -1
	_drag_placed.clear()


# Computes the straight (and, when started on a belt, corner) placement for the
# current drag and sends place orders for any tiles not yet placed this drag.
func _update_belt_drag() -> void:
	if not _drag_active:
		return
	var cur: Vector2i = layer.world_to_tile(get_global_mouse_position())
	var d := cur - _drag_start_tile

	if _drag_started_on_belt:
		_update_belt_drag_from_belt(d, cur)
		return

	# Plain straight line: lock the dominant axis once so it lays cleanly.
	if not _drag_axis_locked and d != Vector2i.ZERO:
		_drag_axis_locked = true
		_drag_axis = 0 if absi(d.x) >= absi(d.y) else 1
	if not _drag_axis_locked:
		return
	var drag_dir: int
	if _drag_axis == 0:
		drag_dir = 1 if (cur.x - _drag_start_tile.x) > 0 else 3
	else:
		drag_dir = 2 if (cur.y - _drag_start_tile.y) > 0 else 0
	var straight: Array = []
	_append_run(straight, _drag_start_tile, drag_dir, cur)
	_place(straight)


# When the drag started on an existing belt, follow the cursor's direction each
# frame: if it travels mostly along the belt it continues straight out its end,
# otherwise it turns, with the corner replacing the belt we started on.
func _update_belt_drag_from_belt(d: Vector2i, cur: Vector2i) -> void:
	if d == Vector2i.ZERO:
		return
	var horizontal := _horizontal(_drag_seed_dir)
	var along := absi(d.x) if horizontal else absi(d.y)
	var perp := absi(d.y) if horizontal else absi(d.x)
	if along > 0 and along >= perp:
		# Continue straight out the seed belt's end.
		var straight: Array = []
		_append_run(straight, _drag_start_tile + _dir_offset(_drag_seed_dir), _drag_seed_dir, cur)
		_place(straight)
		return

	# Turn perpendicular: the corner takes the place of the belt we started on,
	# accepting from the same side the original belt did.
	var in_dir := (_drag_seed_dir + 2) % 4
	var ddir := _perp_dir(_drag_seed_dir, d)
	var flipped := ((ddir + 3) % 4) == in_dir
	var key := _drag_start_tile
	if not _drag_placed.has(key):
		_drag_placed[key] = true
		if _drag_seed_id >= 0:
			NetworkManager.remove(_drag_seed_id)
		NetworkManager.place("belt_turn", key.x, key.y, ddir, flipped)
	var run: Array = []
	_append_run(run, _drag_start_tile + _dir_offset(ddir), ddir, cur)
	_place(run)


# Sends a place order for every placement not already sent during this drag.
func _place(placements: Array) -> void:
	for p in placements:
		var key: Vector2i = p["tile"]
		if _drag_placed.has(key):
			continue
		_drag_placed[key] = true
		NetworkManager.place(p["type"], key.x, key.y, p["dir"], p.get("flipped", false))


# Appends `count` straight belt placements starting at `from` and stepping in
# `dir`, extending at least until the line of tiles reaches `cur`.
func _append_run(placements: Array, from: Vector2i, dir: int, cur: Vector2i) -> void:
	var off := _dir_offset(dir)
	var fc := _coord(dir, from)
	var cc := _coord(dir, cur)
	var count := maxi(1, absi(cc - fc))
	var t := from
	for i in count:
		placements.append({"tile": t, "type": "belt", "dir": dir})
		t += off


func _dir_offset(dir: int) -> Vector2i:
	match dir:
		0:
			return Vector2i(0, -1)
		1:
			return Vector2i(1, 0)
		2:
			return Vector2i(0, 1)
		_:
			return Vector2i(-1, 0)


func _coord(dir: int, t: Vector2i) -> int:
	return t.x if _horizontal(dir) else t.y


func _horizontal(dir: int) -> bool:
	return dir == 1 or dir == 3


# The perpendicular direction to seed_dir that the drag vector heads toward.
func _perp_dir(seed_dir: int, d: Vector2i) -> int:
	if _horizontal(seed_dir):
		return 2 if d.y > 0 else 0
	else:
		return 1 if d.x > 0 else 3


func _update_preview() -> void:
	if _selected == "" or NetworkManager.player_id == "":
		layer.clear_preview()
		return
	var tile: Vector2i = layer.world_to_tile(get_global_mouse_position())
	layer.set_preview(_selected, tile, _rotation, NetworkManager.player_id)


func _unhandled_key_input(event: InputEvent) -> void:
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_F12:
			_set_debug(not _debug_visible)
		elif event.keycode == KEY_F11:
			_show_grid = not _show_grid
			layer.set_show_grid(_show_grid)
		elif event.keycode == KEY_ENTER and not console.has_focus():
			_show_console()
		elif event.keycode == KEY_ESCAPE and console.has_focus():
			_hide_console()
		elif event.keycode == KEY_R and _selected != "":
			_rotation = (_rotation + 1) % 4
			_update_preview()


func _set_debug(on: bool) -> void:
	_debug_visible = on
	hud_label.visible = on


func _zoom(factor: float) -> void:
	var new_zoom: Vector2 = camera.zoom * factor
	new_zoom = new_zoom.clampf(ZOOM_MIN, ZOOM_MAX)
	if new_zoom == camera.zoom:
		return
	var mouse_world := get_global_mouse_position()
	var screen_offset: Vector2 = get_viewport().get_mouse_position() - get_viewport_rect().size * 0.5
	camera.zoom = new_zoom
	camera.position = mouse_world - screen_offset / new_zoom


func _on_welcome(msg: Dictionary) -> void:
	_buildings.clear()
	for b in msg.get("content", {}).get("buildings", []):
		_buildings[b.get("id", "")] = b
	layer.set_content(msg.get("content", {}))
	layer.apply_snapshot(msg.get("snapshot", {}))
	_build_palette()
	_add_chat("system", "joined world '%s' as %s" % [msg.get("text", ""), msg.get("player_name", "")])


func _on_snapshot(msg: Dictionary) -> void:
	layer.apply_snapshot(msg.get("snapshot", {}))


func _on_diff(msg: Dictionary) -> void:
	layer.apply_diff(msg.get("diff", {}))


func _on_disconnected() -> void:
	hud_label.text = "disconnected - retrying in 2s..."


func _on_chat_received(msg: Dictionary) -> void:
	_add_chat(str(msg.get("player_name", "?")), str(msg.get("text", "")))


func _on_system_received(msg: Dictionary) -> void:
	_add_chat("system", str(msg.get("text", "")))


func _add_chat(who: String, text: String) -> void:
	var line := "%s: %s" % [who, text]
	_chat_lines.push_back(line)
	if _chat_lines.size() > 9:
		_chat_lines.pop_front()
	chat_log.text = "\n".join(_chat_lines)


func _build_palette() -> void:
	for c in palette.get_children():
		c.queue_free()

	var remove_btn := Button.new()
	remove_btn.text = "remove"
	remove_btn.toggle_mode = true
	remove_btn.set_meta("id", "remove")
	remove_btn.toggled.connect(_on_tool_toggled.bind("remove"))
	palette.add_child(remove_btn)

	for id in _buildings:
		var btn := Button.new()
		var def: Dictionary = _buildings[id]
		btn.text = "%s  [%s]" % [def.get("name", id), _cost_text(def.get("cost", {}))]
		btn.toggle_mode = true
		btn.set_meta("id", id)
		btn.toggled.connect(_on_tool_toggled.bind(id))
		palette.add_child(btn)

	palette.visible = true


func _on_tool_toggled(pressed: bool, id: String) -> void:
	if not pressed:
		if (_remove_mode and id == "remove") or (_selected == id and not _remove_mode):
			_selected = ""
			_remove_mode = false
			layer.clear_preview()
		return
	_remove_mode = id == "remove"
	_selected = "" if _remove_mode else id
	_rotation = 1
	for c in palette.get_children():
		if c is Button and c.has_meta("id") and c.get_meta("id") != id and c.button_pressed:
			c.set_pressed_no_signal(false)
	_update_preview()


func _cost_text(cost: Dictionary) -> String:
	var parts: Array[String] = []
	for res in cost:
		parts.append("%s %d" % ["%s" % res, cost[res]])
	return ", ".join(parts)


func _materials_text(player: Dictionary) -> String:
	if player.is_empty():
		return ""
	var mats: Dictionary = player.get("resources", {})
	if mats.is_empty():
		return "no materials"
	var lines: Array[String] = []
	for res in mats:
		var def: Dictionary = layer.resource_def(res)
		var name: String = str(def.get("name", res))
		lines.append("%s: %d" % [name, int(mats[res])])
	lines.sort()
	return "\n".join(lines)


func _show_console() -> void:
	console.visible = true
	console_hint.visible = false
	console.grab_focus()


func _hide_console() -> void:
	console.visible = false
	console_hint.visible = true


func _on_console_submitted(text: String) -> void:
	if text.strip_edges() != "":
		NetworkManager.chat(text)
		if not text.begins_with("/"):
			var who := "you" if NetworkManager.player_id != "" else str(NetworkManager.player_id)
			_add_chat(who, text)
	console.clear()
	_hide_console()
