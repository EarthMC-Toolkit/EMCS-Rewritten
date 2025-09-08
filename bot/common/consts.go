package common

const DEFAULT_TOWN_BOARD = "/town set board [msg]"

type EMCMap = string

var SUPPORTED_MAPS = struct {
	AURORA EMCMap
}{
	AURORA: "aurora",
}

type Emoji = string

var EMOJIS = struct {
	ARROW_DOWN_RED Emoji
	ARROW_UP_GREEN Emoji
	SHIELD_RED     Emoji
	SHIELD_GREEN   Emoji
	GOLD_INGOT     Emoji
	CHUNK          Emoji
}{
	ARROW_DOWN_RED: "<:arrow_down_red:1413371841170374766>",
	ARROW_UP_GREEN: "<:arrow_up_green:1413371794076991680>",
	SHIELD_RED:     "<:shield_red:1407578605134807113>",
	SHIELD_GREEN:   "<:shield_green:1407578592023412897>",
	GOLD_INGOT:     "<:gold_ingot:1408321088823492728>",
	CHUNK:          "<:chunk:1408321296894398504>",
}
