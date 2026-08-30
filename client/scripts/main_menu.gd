extends Control

@onready var _name_edit: LineEdit = %NameEdit
@onready var _play_btn: Button = %PlayButton


func _ready() -> void:
	if Settings.is_dev():
		get_tree().call_deferred("change_scene_to_file", "res://scenes/main.tscn")
		return
	_play_btn.pressed.connect(_on_browse)
	_name_edit.text_changed.connect(_on_name_changed)
	_name_edit.grab_focus()


func _on_name_changed(text: String) -> void:
	Settings.player_name = text.strip_edges()
	Settings.save()


func _on_browse() -> void:
	if Settings.player_name == "":
		_name_edit.text = "Player"
		Settings.player_name = "Player"
		Settings.save()
	get_tree().change_scene_to_file("res://scenes/browser.tscn")