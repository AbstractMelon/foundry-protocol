extends Node

signal connected(ws_url)
signal disconnected
signal welcome_received(msg)
signal snapshot_received(msg)
signal diff_received(msg)
signal chat_received(msg)
signal system_received(msg)
signal servers_fetched(gateway_url, servers, error_text)

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
	auto_connect = Settings.is_dev()
	ws_url = ProjectSettings.get_setting("network/ws_url", "ws://localhost:8090/ws")
	player_name = Settings.player_name
	var env_url := OS.get_environment("FOUNDRY_WS_URL")
	if env_url != "":
		ws_url = env_url
	var url_arg: String = str(Settings.custom_user_args.get("ws", ""))
	if url_arg != "":
		ws_url = url_arg
	connected.connect(_on_socket_connected)
	if auto_connect:
		connect_to_server(ws_url)


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
	if _socket == null:
		return
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
	return _socket != null and _socket.get_ready_state() == WebSocketPeer.STATE_OPEN


func is_connect_pending() -> bool:
	return _socket != null and _socket.get_ready_state() != WebSocketPeer.STATE_CLOSED


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


func place(building_type: String, x: int, y: int, dir := 1, flipped := false) -> void:
	send_message({"type": "place_building", "building_type": building_type, "tile_x": x, "tile_y": y, "dir": dir, "flipped": flipped})


func remove(entity_id: int) -> void:
	send_message({"type": "remove_building", "entity_id": entity_id})


func chat(text: String) -> void:
	send_message({"type": "chat", "text": text})


func normalize_gateway_url(url: String) -> String:
	var u := url.strip_edges()
	if u == "" or u == "http://" or u == "https://":
		return ""
	if not u.begins_with("http://") and not u.begins_with("https://"):
		u = "http://" + u
	u = u.trim_suffix("/")
	return u


func fetch_servers(gateway_url: String) -> void:
	var base := normalize_gateway_url(gateway_url)
	if base == "":
		servers_fetched.emit(gateway_url, [], "invalid gateway URL")
		return
	var http := HTTPRequest.new()
	http.timeout = 8.0
	add_child(http)
	http.request_completed.connect(_on_servers_fetched.bind(http, base))
	var err := http.request(base + "/servers")
	if err != OK:
		servers_fetched.emit(base, [], "failed to reach gateway")
		http.queue_free()


func _on_servers_fetched(result: int, code: int, _headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest, base: String) -> void:
	http.queue_free()
	if result != HTTPRequest.RESULT_SUCCESS:
		servers_fetched.emit(base, [], "failed to reach gateway (HTTP error %d)" % result)
		return
	if code != 200:
		servers_fetched.emit(base, [], "gateway returned HTTP %d" % code)
		return
	var parsed: Variant = JSON.parse_string(body.get_string_from_utf8())
	if typeof(parsed) != TYPE_DICTIONARY:
		servers_fetched.emit(base, [], "gateway returned invalid response")
		return
	var servers: Array = parsed.get("servers", [])
	servers_fetched.emit(base, servers, "")
