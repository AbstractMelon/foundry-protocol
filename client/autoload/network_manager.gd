extends Node

signal connected(ws_url)
signal disconnected
signal welcome_received(msg)
signal snapshot_received(msg)
signal diff_received(msg)
signal chat_received(msg)
signal system_received(msg)

const RECONNECT_DELAY := 2.0
const SOCKET_BUFFER_SIZE := 16 * 1024 * 1024

var auto_connect := false
var ws_url := "ws://localhost:8090/ws"
var player_name := "Dev"
var player_id := ""
var last_tick := 0

var _socket: WebSocketPeer
var _connected := false
var _retry := 0.0


func _ready() -> void:
	auto_connect = ProjectSettings.get_setting("network/auto_connect", false)
	ws_url = ProjectSettings.get_setting("network/ws_url", "ws://localhost:8090/ws")
	player_name = ProjectSettings.get_setting("network/player_name", "Dev")
	var env_url := OS.get_environment("FOUNDRY_WS_URL")
	if env_url != "":
		ws_url = env_url
	connected.connect(_on_socket_connected)
	if auto_connect:
		connect_to_server(ws_url)
	else:
		_socket = make_socket(ws_url)


func make_socket(url: String) -> WebSocketPeer:
	var peer := WebSocketPeer.new()
	peer.set_inbound_buffer_size(SOCKET_BUFFER_SIZE)
	peer.set_outbound_buffer_size(SOCKET_BUFFER_SIZE)
	peer.set_max_queued_packets(8192)
	peer.connect_to_url(url)
	return peer


func _on_socket_connected(_url: String) -> void:
	send_hello(player_name)


func _process(delta: float) -> void:
	var state := _socket.get_ready_state()
	if state == WebSocketPeer.STATE_OPEN:
		_socket.poll()
		if not _connected:
			_connected = true
			connected.emit(ws_url)
		while _socket.get_available_packet_count() > 0:
			_handle_packet(_socket.get_packet())
	elif state == WebSocketPeer.STATE_CONNECTING:
		_socket.poll()
	elif state == WebSocketPeer.STATE_CLOSED:
		if _connected:
			_connected = false
			disconnected.emit()
		if auto_connect:
			_retry += delta
			if _retry >= RECONNECT_DELAY:
				_retry = 0.0
				_socket = make_socket(ws_url)


func connect_to_server(url: String) -> void:
	ws_url = url
	_socket = make_socket(url)
	_connected = false
	_retry = 0.0
	if _socket.get_ready_state() == WebSocketPeer.STATE_CLOSED:
		push_warning("WebSocket connect failed for %s" % ws_url)


func is_connected_to_server() -> bool:
	return _socket.get_ready_state() == WebSocketPeer.STATE_OPEN


func get_ping() -> float:
	if is_connected_to_server() and _socket.has_method("get_current_ping"):
		return float(_socket.get_current_ping())
	return 0.0


func _handle_packet(data: PackedByteArray) -> void:
	var parsed: Variant = JSON.parse_string(data.get_string_from_utf8())
	if typeof(parsed) != TYPE_DICTIONARY:
		return
	var msg: Dictionary = parsed
	match msg.get("type", ""):
		"welcome":
			player_id = msg.get("player_id", "")
			last_tick = msg.get("snapshot", {}).get("tick", 0)
			welcome_received.emit(msg)
		"snapshot":
			last_tick = msg.get("snapshot", {}).get("tick", 0)
			snapshot_received.emit(msg)
		"diff":
			last_tick = msg.get("diff", {}).get("tick", last_tick)
			diff_received.emit(msg)
		"chat_received":
			chat_received.emit(msg)
		"system":
			system_received.emit(msg)


func send_message(dict: Dictionary) -> void:
	if is_connected_to_server():
		_socket.send_text(JSON.stringify(dict))


func send_hello(name: String) -> void:
	send_message({"type": "hello", "name": name})


func place(building_type: String, x: int, y: int, dir := 1) -> void:
	send_message({"type": "place_building", "building_type": building_type, "tile_x": x, "tile_y": y, "dir": dir})


func remove(entity_id: int) -> void:
	send_message({"type": "remove_building", "entity_id": entity_id})


func chat(text: String) -> void:
	send_message({"type": "chat", "text": text})