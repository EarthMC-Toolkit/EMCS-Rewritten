package common

const DEFAULT_TOWN_BOARD = "/town set board [msg]"

type Emoji = string

var EMOJIS = struct {
	SHIELD_RED   Emoji
	SHIELD_GREEN Emoji
	GOLD_INGOT   Emoji
	CHUNK        Emoji
}{
	SHIELD_RED:   "<:shield_red:1407578605134807113>",
	SHIELD_GREEN: "<:shield_green:1407578592023412897>",
	GOLD_INGOT:   "<:gold_ingot:1408321088823492728>",
	CHUNK:        "<:chunk:1408321296894398504>",
}
