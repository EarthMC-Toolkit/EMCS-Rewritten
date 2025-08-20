package oapi

import (
	"emcsrw/oapi/objs"
	"emcsrw/utils"
	"fmt"
)

func Resident(identifier string) (objs.PlayerInfo, error) {
	endpoint := fmt.Sprintf("/players/%s", identifier)
	resident, err := utils.OAPIGetRequest[objs.PlayerInfo](endpoint)

	if err != nil {
		return objs.PlayerInfo{}, err
	}

	return resident, nil
}
