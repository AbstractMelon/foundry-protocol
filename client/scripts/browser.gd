extends Control

const TITLE_GATEWAYS := "Server Gateways"
const TITLE_SERVERS := "Servers on "

const GatewayRowScene := preload("res://scenes/ui/gateway_row.tscn")
const ServerCardScene := preload("res://scenes/ui/server_card.tscn")

var _state := "gateways"
var _active_gateway := ""

@onready var _back_btn: Button = %BackButton
@onready var _title: Label = %Title
@onready var _player: Label = %PlayerLabel
@onready var _content: VBoxContainer = %Content
@onready var _status: Label = %Status
@onready var _add_section: VBoxContainer = %AddGatewaySection
@onready var _new_gateway_edit: LineEdit = %NewGatewayEdit
@onready var _add_gateway_btn: Button = %AddGatewayBtn
@onready var _section: Label = %SectionLabel
@onready var _back_to_list: Button = %BackToListButton


func _ready() -> void:
	NetworkManager.servers_fetched.connect(_on_servers_fetched)
	_back_btn.pressed.connect(_on_back)
	_back_to_list.pressed.connect(_on_back_to_gateways)
	_add_gateway_btn.pressed.connect(_add_gateway)
	_new_gateway_edit.text_submitted.connect(_on_add_submitted)
	_refresh()


func _clear_content() -> void:
	for child in _content.get_children():
		child.queue_free()


func _refresh() -> void:
	_clear_content()
	_status.text = ""
	_player.text = "playing as %s" % Settings.player_name
	match _state:
		"gateways":
			_add_section.visible = true
			_back_to_list.visible = false
			_section.text = "SAVED GATEWAYS"
			_title.text = TITLE_GATEWAYS
			_back_btn.text = "< Back"
			_build_gateway_list()
		"servers":
			_add_section.visible = false
			_back_to_list.visible = true
			_section.text = "Contacting gateway..."
			_title.text = TITLE_SERVERS + _active_gateway
			_back_btn.text = "< Gateways"
			_build_server_list()


func _on_back() -> void:
	if _state == "servers":
		_on_back_to_gateways()
	else:
		get_tree().change_scene_to_file("res://scenes/main_menu.tscn")


func _on_back_to_gateways() -> void:
	_state = "gateways"
	_refresh()


func _build_gateway_list() -> void:
	if Settings.gateways.is_empty():
		_content.add_child(_empty_label("No gateways yet. Add one above."))
		return
	for i in Settings.gateways.size():
		var row := GatewayRowScene.instantiate()
		_content.add_child(row)
		row.setup(str(Settings.gateways[i].get("url", "")), i)
		row.browse_requested.connect(_on_browse_gateway)
		row.remove_requested.connect(_on_remove_gateway)


func _build_server_list() -> void:
	NetworkManager.fetch_servers(_active_gateway)


func _on_servers_fetched(gateway_url: String, servers: Array, error_text: String) -> void:
	if _state != "servers" or gateway_url != _active_gateway:
		return
	_clear_content()
	_status.text = error_text
	if error_text != "":
		_section.text = "Could not reach gateway"
		return
	_section.text = "%d server(s) found" % servers.size()
	if servers.is_empty():
		_content.add_child(_empty_label("This gateway has no servers registered."))
		return
	for s in servers:
		var card := ServerCardScene.instantiate()
		_content.add_child(card)
		card.setup(s)
		card.join_pressed.connect(_on_join)


func _empty_label(text: String) -> Label:
	var l := Label.new()
	l.text = text
	l.add_theme_color_override("font_color", Color(0.55, 0.6, 0.58))
	return l


func _on_add_submitted(_text: String) -> void:
	_add_gateway()


func _add_gateway() -> void:
	if Settings.add_gateway(_new_gateway_edit.text):
		_new_gateway_edit.text = ""
		_refresh()
	else:
		_status.text = "Could not add that gateway (empty, duplicate, or malformed?)"


func _on_remove_gateway(index: int) -> void:
	Settings.remove_gateway(index)
	_refresh()


func _on_browse_gateway(url: String) -> void:
	_active_gateway = NetworkManager.normalize_gateway_url(url)
	_state = "servers"
	_refresh()


func _on_join(s: Dictionary) -> void:
	var ws_url: String = str(s.get("ws_url", "")).strip_edges()
	if ws_url == "":
		_status.text = "That server has no ws_url, cannot join."
		return
	if not ws_url.begins_with("ws://") and not ws_url.begins_with("wss://"):
		ws_url = "ws://" + ws_url
	NetworkManager.ws_url = ws_url
	NetworkManager.player_name = Settings.player_name
	NetworkManager.connect_to_server(NetworkManager.ws_url)
	get_tree().change_scene_to_file("res://scenes/main.tscn")