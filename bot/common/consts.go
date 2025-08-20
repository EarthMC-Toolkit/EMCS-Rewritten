package common

type Emoji = string

var SHIELD_EMOJIS = struct {
	RED   Emoji
	GREEN Emoji
}{
	RED:   "<:shield_red:1407578605134807113>",
	GREEN: "<:shield_green:1407578592023412897>",
}

const DEFAULT_TOWN_BOARD = "/town set board [msg]"
