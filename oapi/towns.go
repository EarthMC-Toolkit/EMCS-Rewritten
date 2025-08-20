package oapi

import (
	"emcsrw/oapi/objs"
	"emcsrw/utils"
)

func QueryTowns(identifiers ...string) ([]objs.TownInfo, error) {
	return utils.OAPIPostRequest[[]objs.TownInfo]("/towns", NewNamesQuery(identifiers...))
}
