package common

const DEFAULT_TOWN_BOARD = "/town set board [msg]"

type Emoji = string

const GOLD_INGOT_EMOJI Emoji = "<:gold_ingot:1408321088823492728>"

var SHIELD_EMOJIS = struct {
	RED   Emoji
	GREEN Emoji
}{
	RED:   "<:shield_red:1407578605134807113>",
	GREEN: "<:shield_green:1407578592023412897>",
}
