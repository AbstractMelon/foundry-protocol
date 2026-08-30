extends PanelContainer

signal join_pressed(server: Dictionary)

@onready var _name_label: Label = %Name
@onready var _desc_label: Label = %Description
@onready var _meta_label: Label = %Meta
@onready var _join_btn: Button = %JoinButton

var _server: Dictionary = {}


func _ready() -> void:
	_join_btn.pressed.connect(_on_join)


func setup(server: Dictionary) -> void:
	_server = server
	_name_label.text = str(server.get("name", "Unnamed"))
	_desc_label.text = str(server.get("description", ""))
	_desc_label.visible = _desc_label.text != ""
	var owner: String = str(server.get("owner", ""))
	var maxp: int = int(server.get("max_players", 0))
	_meta_label.text = "%s  |  max players: %s" % [owner, "unknown" if maxp <= 0 else str(maxp)]


func _on_join() -> void:
	join_pressed.emit(_server)