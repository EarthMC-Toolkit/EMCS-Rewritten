package oapi

import (
	"emcsrw/utils"
)

type NamesQuery struct {
	Query []string `json:"query"`
}

func NewNamesQuery(names ...string) NamesQuery {
	return NamesQuery{Query: names}
}

func QueryPlayers(identifiers ...string) ([]PlayerInfo, error) {
	return utils.OAPIPostRequest[[]PlayerInfo]("/players", NewNamesQuery(identifiers...))
}

func QueryTowns(identifiers ...string) ([]TownInfo, error) {
	return utils.OAPIPostRequest[[]TownInfo]("/towns", NewNamesQuery(identifiers...))
}

func QueryNations(identifiers ...string) ([]NationInfo, error) {
	return utils.OAPIPostRequest[[]NationInfo]("/nations", NewNamesQuery(identifiers...))
}

func QueryServer() (RawServerInfoV3, error) {
	return utils.OAPIGetRequest[RawServerInfoV3]("")
}
