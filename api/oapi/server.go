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

type RawServerInfoV3 struct {
	Version    string           `json:"version"`
	MoonPhase  MoonPhaseV3      `json:"moonPhase"`
	Timestamps ServerTimestamps `json:"timestamps"`
	Status     ServerStatus     `json:"status"`
	Stats      ServerStats      `json:"stats"`
	VoteParty  ServerVoteParty  `json:"voteParty"`
}
