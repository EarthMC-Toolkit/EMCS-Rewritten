package oapi

import (
	"encoding/json"
	"fmt"
)

type MoonPhaseV3 string

func (m *MoonPhaseV3) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case
		string(MoonPhaseFirstQuarter),
		string(MoonPhaseFullMoon),
		string(MoonPhaseLastQuarter),
		string(MoonPhaseNewMoon),
		string(MoonPhaseWaningCrescent),
		string(MoonPhaseWaningGibbous),
		string(MoonPhaseWaxingCrescent),
		string(MoonPhaseWaxingGibbous):

		*m = MoonPhaseV3(s)

		return nil
	default:
		return fmt.Errorf("invalid moon phase: %s", s)
	}
}

const (
	MoonPhaseFirstQuarter   MoonPhaseV3 = "FIRST_QUARTER"
	MoonPhaseFullMoon       MoonPhaseV3 = "FULL_MOON"
	MoonPhaseLastQuarter    MoonPhaseV3 = "LAST_QUARTER"
	MoonPhaseNewMoon        MoonPhaseV3 = "NEW_MOON"
	MoonPhaseWaningCrescent MoonPhaseV3 = "WANING_CRESCENT"
	MoonPhaseWaningGibbous  MoonPhaseV3 = "WANING_GIBBOUS"
	MoonPhaseWaxingCrescent MoonPhaseV3 = "WAXING_CRESCENT"
	MoonPhaseWaxingGibbous  MoonPhaseV3 = "WAXING_GIBBOUS"
)

type ServerStats struct {
	Time             int64 `json:"time"`
	FullTime         int64 `json:"fullTime"`
	MaxPlayers       int   `json:"maxPlayers"`
	NumOnlinePlayers int   `json:"numOnlinePlayers"`
	NumOnlineNomads  int   `json:"numOnlineNomads"`
	NumResidents     int   `json:"numResidents"`
	NumNomads        int   `json:"numNomads"`
	NumTowns         int   `json:"numTowns"`
	NumTownBlocks    int   `json:"numTownBlocks"`
	NumNations       int   `json:"numNations"`
	NumQuarters      int   `json:"numQuarters"`
	NumCuboids       int   `json:"numCuboids"`
}

type ServerTimestamps struct {
	NewDayTime      int64 `json:"newDayTime"`
	ServerTimeOfDay int64 `json:"serverTimeOfDay"`
}

type ServerStatus struct {
	HasStorm     bool `json:"hasStorm"`
	IsThundering bool `json:"isThundering"`
}

type ServerVoteParty struct {
	Target       int `json:"target"`
	NumRemaining int `json:"numRemaining"`
}

type ServerInfo struct {
	Version    string           `json:"version"`
	MoonPhase  MoonPhaseV3      `json:"moonPhase"`
	Timestamps ServerTimestamps `json:"timestamps"`
	Status     ServerStatus     `json:"status"`
	Stats      ServerStats      `json:"stats"`
	VoteParty  ServerVoteParty  `json:"voteParty"`
}

type ServerPlayerStats map[string]uint // Represents list below

// "enchant_item": 3738350,
// "strider_one_cm": 1398277463,
// "time_since_death": 38729078969,
// "open_barrel": 40638053,
// "swim_one_cm": 30165631394,
// "damage_blocked_by_shield": 87902904,
// "interact_with_furnace": 21185916,
// "damage_dealt_resisted": 83519,
// "damage_dealt": 6917901117,
// "sneak_time": 8915286184,
// "time_since_rest": 32227388229,
// "bell_ring": 213252,
// "interact_with_loom": 296689,
// "damage_absorbed": 63718566,
// "eat_cake_slice": 187349,
// "play_noteblock": 2731536,
// "use_cauldron": 893473,
// "climb_one_cm": 17353477295,
// "raid_win": 2,
// "animals_bred": 2031263,
// "target_hit": 67284,
// "sprint_one_cm": 718824101479,
// "tune_noteblock": 40706367,
// "total_world_time": 305781486463,
// "interact_with_blast_furnace": 3461149,
// "traded_with_villager": 350178,
// "talked_to_villager": 2275489,
// "walk_one_cm": 677102898385,
// "play_record": 158136,
// "pot_flower": 122580,
// "interact_with_crafting_table": 17012249,
// "inspect_hopper": 5195251,
// "interact_with_beacon": 250466,
// "drop_count": 72177517,
// "player_kills": 498099,
// "clean_banner": 19842,
// "interact_with_smithing_table": 336598,
// "interact_with_brewingstand": 14638651,
// "open_shulker_box": 244,
// "inspect_dispenser": 1002024,
// "mob_kills": 151557458,
// "jump": 3674772119,
// "minecart_one_cm": 4403717516,
// "open_chest": 165418566,
// "damage_dealt_absorbed": 58649744,
// "pig_one_cm": 79829885,
// "fly_one_cm": 897963509433,
// "raid_trigger": 2,
// "play_time": 305781486463,
// "clean_armor": 733,
// "crouch_one_cm": 37672156686,
// "fall_one_cm": 110929565773,
// "walk_under_water_one_cm": 37141593494,
// "interact_with_stonecutter": 1176501,
// "interact_with_cartography_table": 132146,
// "leave_game": 8583558,
// "interact_with_smoker": 725864,
// "damage_resisted": 149962261,
// "horse_one_cm": 11098411486,
// "boat_one_cm": 164222511303,
// "walk_on_water_one_cm": 25675360613,
// "sleep_in_bed": 262340,
// "fish_caught": 11906262,
// "interact_with_grindstone": 1082506,
// "interact_with_campfire": 665809,
// "interact_with_lectern": 831592,
// "open_enderchest": 20438857,
// "inspect_dropper": 1480910,
// "interact_with_anvil": 1925246,
// "damage_taken": 1820311317,
// "trigger_trapped_chest": 174842,
// "fill_cauldron": 84705,
// "clean_shulker_box": 1,
// "aviate_one_cm": 127129105,
// "deaths": 1491338
