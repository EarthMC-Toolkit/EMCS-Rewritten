package shared

const NAMEMC_URL = "https://namemc.com/profile/"

const DEFAULT_TOWN_BOARD = "/town set board [msg]"
const DEFAULT_ABOUT = "/res set about [msg]"

const FLAGS_CHANNEL_ID = "966372674236481606"

type EMCMap = string

var ACTIVE_MAP = SUPPORTED_MAPS.AURORA
var SUPPORTED_MAPS = struct {
	AURORA EMCMap
}{
	AURORA: "aurora",
}

type Emoji = string

var EMOJIS = struct {
	CIRCLE_CHECK   Emoji
	CIRCLE_CROSS   Emoji
	ENTRY          Emoji
	EXIT           Emoji
	ARROW_DOWN_RED Emoji
	ARROW_UP_GREEN Emoji
	SHIELD_RED     Emoji
	SHIELD_GREEN   Emoji
	GOLD_INGOT     Emoji
	CHUNK          Emoji
}{
	CIRCLE_CHECK:   "<:circle_check:1448776921994367078>",
	CIRCLE_CROSS:   "<:circle_cross:1448776923093139579>",
	ENTRY:          "<:entry:1448776924653555904>",
	EXIT:           "<:exit:1448776925953654975>",
	ARROW_DOWN_RED: "<:arrow_down_red:1448776656197128202>",
	ARROW_UP_GREEN: "<:arrow_up_green:1448776583249924381>",
	SHIELD_RED:     "<:shield_red:1448776535426338877>",
	SHIELD_GREEN:   "<:shield_green:1448775998823727144>",
	GOLD_INGOT:     "<:gold_ingot:1349202219387322448>",
	CHUNK:          "<:chunk:1349202301545349150>",
}

var ACTION_SPEEDS = struct {
	SNEAK  float32
	WALK   float32
	SPRINT float32
	BOAT   float32
}{
	SNEAK:  1.295,
	WALK:   4.317,
	SPRINT: 5.612,
	BOAT:   8.0,
}
