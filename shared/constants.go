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
	CIRCLE_CHECK:   "<:circle_check:1442466686794989659>",
	CIRCLE_CROSS:   "<:circle_cross:1442466697201192960>",
	ENTRY:          "<:entry:1430368475309936721>",
	EXIT:           "<:exit:1430368497996791901>",
	ARROW_DOWN_RED: "<:arrow_down_red:1413371841170374766>",
	ARROW_UP_GREEN: "<:arrow_up_green:1413371794076991680>",
	SHIELD_RED:     "<:shield_red:1407578605134807113>",
	SHIELD_GREEN:   "<:shield_green:1407578592023412897>",
	GOLD_INGOT:     "<:gold_ingot:1408321088823492728>",
	CHUNK:          "<:chunk:1408321296894398504>",
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

// TODO: Acquire staff ids from Fruitloopins repo instead?
// var STAFF_OWNER_IDS = [...]string{
// 	"7395d056-536a-4cf3-9c96-6c7a7df6897a",
// }

// var STAFF_ADMIN_IDS = [...]string{
// 	"f17d77ab-aed4-44e7-96ef-ec9cd473eda3",
// 	"ea798ca9-1192-4334-8ef5-f098bdc9cb39",
// }

// var STAFF_DEV_IDS = [...]string{
// 	"fed0ec4a-f1ad-4b97-9443-876391668b34",
// 	"e25ad129-fe8a-4306-b1af-1dee1ff59841",
// }

// var STAFF_STAFF_MANAGER_IDS = [...]string{
// 	"3947e217-95ae-4952-93b7-6b4004b54f40",
// 	"8f90a970-90de-407d-b4e7-0d9fde10f51a",
// 	"8391474f-4b57-412a-a835-96bd2c253219",
// }

// var STAFF_MOD_IDS = [...]string{
// 	"379f1b2b-0b53-4ea4-b0b2-14b6e4777983",
// 	"be66697b-bde8-4d70-b339-4ba15b390fa9",
// 	"9449b781-a702-484a-934d-9ed351bcebd4",
// 	"d60547bd-83d4-41f4-bc91-9151385ea527",
// 	"bb5cfe53-4773-47d4-94b3-79bb6253885f",
// 	"d32c9786-17e6-43a2-8f83-8f95a6655893",
// 	"7b589f55-c5e2-41b7-89fc-13eb8781058e",
// 	"3c84a17b-1eb5-4201-a8e2-391bde3155b0",
// 	"686b1bfa-38de-4eb9-9628-43ed3b56b76e",
// 	"5b8274bf-b162-4336-85a0-48f9d5380a78",
// 	"67f557b3-c57d-4c51-903c-9193480a1247",
// 	"c6e76e78-a9eb-44c1-9aa2-2f9b48693528",
// 	"24669297-3f32-48e5-b909-4c4aedd8640b",
// 	"8c7f58b3-c2bb-4a66-bf22-0b914331c006",
// }

// var STAFF_HELPER_IDS = [...]string{
// 	"e378b23d-6a5e-4cf3-8dc6-64ed8761df36",
// 	"54942c49-465c-481e-a8a0-a8e5b6417082",
// 	"fe89d9a7-8927-4d02-b09e-978a3a3c9f1d",
// 	"1619c62c-0198-4139-a204-cd767d4b95c5",
// 	"c23e0d78-c2ea-4477-93b5-440ee36f13a7",
// 	"1c3e1b8f-155c-47ca-99ac-fe8b96e834cc",
// 	"04369303-e2c9-4890-94de-2bbbb2add7e5",
// 	"03e40235-fc6f-457d-9d06-ecf641a8bb18",
// 	"23eb774a-4837-4568-839d-50be6302c545",
// 	"e6b204ec-1510-4f92-a759-c892c9b70c60",
// }
