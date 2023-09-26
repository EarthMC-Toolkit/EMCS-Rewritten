package oapi

import (
	"emcsrw/utils"
	"emcsrw/oapi/structs"
	"fmt"
)

func Town(name string) (structs.TownInfo, error) {
	town, err := utils.JsonRequest[structs.TownInfo](fmt.Sprintf("/towns/%s", name), false)

	if err != nil { 
		return structs.TownInfo{}, err
	}
	
	return town, nil
}