package oapi

import (
	"emcsrw/oapi/objs"
	"emcsrw/utils"
)

func QueryPlayers(identifiers ...string) ([]objs.PlayerInfo, error) {
	return utils.OAPIPostRequest[[]objs.PlayerInfo]("/players", NewNamesQuery(identifiers...))
}

// func Resident(identifier string) (objs.PlayerInfo, error) {
// 	endpoint := fmt.Sprintf("/players", identifier)
// 	resident, err := utils.OAPIGetRequest[objs.PlayerInfo](endpoint)
// 	if err != nil {
// 		return objs.PlayerInfo{}, err
// 	}

// 	return resident, nil
// }
