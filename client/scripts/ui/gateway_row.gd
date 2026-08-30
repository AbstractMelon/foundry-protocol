extends HBoxContainer

signal browse_requested(url: String)
signal remove_requested(index: int)

@onready var _browse_btn: Button = %BrowseButton
@onready var _remove_btn: Button = %RemoveButton

var _url := ""
var _index := -1


func _ready() -> void:
	_browse_btn.pressed.connect(_on_browse)
	_remove_btn.pressed.connect(_on_remove)


func setup(url: String, index: int) -> void:
	_url = url
	_index = index
	_browse_btn.text = url
	_browse_btn.tooltip_text = "Browse this gateway's servers"


func _on_browse() -> void:
	browse_requested.emit(_url)


func _on_remove() -> void:
	remove_requested.emit(_index)