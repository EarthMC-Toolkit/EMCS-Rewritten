package structs

type MoonPhaseV3 string

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

type RawServerInfoV3 struct {
	Version    string `json:"version"`
	MoonPhase  string `json:"moonPhase"`
	Timestamps struct {
		NewDayTime      int64 `json:"newDayTime"`
		ServerTimeOfDay int64 `json:"serverTimeOfDay"`
	} `json:"timestamps"`
	Status struct {
		HasStorm     bool `json:"hasStorm"`
		IsThundering bool `json:"isThundering"`
	} `json:"status"`
	Stats struct {
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
	} `json:"stats"`
	VoteParty struct {
		Target       int `json:"target"`
		NumRemaining int `json:"numRemaining"`
	} `json:"voteParty"`
}

// type ServerWorld struct {
// 	Storming   bool  `json:"hasStorm"`
// 	Thundering bool  `json:"isThundering"`
// 	Time       int16 `json:"time"`
// 	FullTime   int32 `json:"fullTime"`
// }

// type ServerPlayers struct {
// 	Max            int16 `json:"maxPlayers"`
// 	OnlineTownless int16 `json:"numOnlineTownless"`
// 	OnlinePlayers  int16 `json:"numOnlinePlayers"`
// }

// type ServerStats struct {
// 	Residents  int32 `json:"numResidents"`
// 	Townless   int32 `json:"numTownless"`
// 	Towns      int16 `json:"numTowns"`
// 	Nations    int16 `json:"numNations"`
// 	TownBlocks int32 `json:"numTownBlocks"`
// }

// type ServerInfo struct {
// 	World   ServerWorld
// 	Players ServerPlayers
// 	Stats   ServerStats
// }
