package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"fmt"
)

func Town(name string) (structs.TownInfo, error) {
	ep := fmt.Sprintf("/towns/%s", name)
	town, err := utils.OAPIRequest[structs.TownInfo](ep)

	if err != nil {
		return structs.TownInfo{}, err
	}

	return town, nil
}
