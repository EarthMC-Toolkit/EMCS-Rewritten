package oapi

import (
	"emcsrw/utils"
	"emcsrw/oapi/structs"
	"fmt"
)

func Nation(name string) (structs.NationInfo, error) {
	nation, err := utils.JsonRequest[structs.NationInfo](fmt.Sprintf("/nations/%s", name), false)

	if err != nil { 
		return structs.NationInfo{}, err
	}
	
	return nation, nil
}