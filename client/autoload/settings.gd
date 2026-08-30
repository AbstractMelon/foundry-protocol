extends Node

const SAVE_PATH := "user://foundry.json"

var player_name := "Dev"
var gateways: Array = []
var custom_user_args := {}


func _ready() -> void:
	_load_cmdline()
	_load()


func _load_cmdline() -> void:
	var i := 0
	var user_args := OS.get_cmdline_user_args()
	while i < user_args.size():
		var arg := user_args[i]
		if arg.begins_with("--"):
			var key := arg.trim_prefix("--")
			var value := "true"
			if key.contains("="):
				value = key.substr(key.find("=") + 1)
				key = key.substr(0, key.find("="))
			custom_user_args[key] = value
		i += 1
	var name_arg := _arg(["user", "name"])
	if name_arg != "":
		player_name = name_arg


func _arg(keys: Array, fallback := "") -> String:
	for k in keys:
		if custom_user_args.has(str(k)):
			return str(custom_user_args[str(k)])
	return fallback


func is_dev() -> bool:
	return _bool("dev")


func _bool(key: String) -> bool:
	if not custom_user_args.has(key):
		return false
	var v: String = str(custom_user_args[key]).to_lower()
	return v == "true" or v == "1" or v == "yes"


func _load() -> void:
	if not FileAccess.file_exists(SAVE_PATH):
		return
	var f := FileAccess.open(SAVE_PATH, FileAccess.READ)
	if f == null:
		return
	var data: Variant = JSON.parse_string(f.get_as_text())
	f.close()
	if typeof(data) != TYPE_DICTIONARY:
		return
	if data.has("player_name"):
		player_name = str(data["player_name"])
	if typeof(data.get("gateways")) == TYPE_ARRAY:
		gateways = data["gateways"]


func save() -> void:
	var f := FileAccess.open(SAVE_PATH, FileAccess.WRITE)
	if f == null:
		return
	f.store_string(JSON.stringify({
		"player_name": player_name,
		"gateways": gateways,
	}))
	f.close()


func add_gateway(url: String) -> bool:
	var clean := _normalize(url)
	if clean == "":
		return false
	for g in gateways:
		if str(g.get("url", "")).to_lower() == clean.to_lower():
			return false
	gateways.append({
		"url": clean,
		"name": clean,
	})
	save()
	return true


func remove_gateway(index: int) -> void:
	if index >= 0 and index < gateways.size():
		gateways.remove_at(index)
		save()


func _normalize(url: String) -> String:
	var u := url.strip_edges()
	if u == "":
		return ""
	u = u.replace("http://", "").replace("https://", "")
	u = u.replace("ws://", "").replace("wss://", "")
	u = u.replace("/", "")
	return u
