package oapi

import (
	"emcsrw/utils"
	"emcsrw/oapi/structs"
	"fmt"
)

func Town(name string) (structs.TownInfo, error) {
	ep := fmt.Sprintf("/towns/%s", name)
	town, err := utils.OAPIRequest[structs.TownInfo](ep, false)

	if err != nil { 
		return structs.TownInfo{}, err
	}
	
	return town, nil
}