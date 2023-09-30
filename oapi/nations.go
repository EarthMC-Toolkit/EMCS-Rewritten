package oapi

import (
	"emcsrw/utils"
	"emcsrw/oapi/structs"
	"fmt"
)

func Nation(name string) (structs.NationInfo, error) {
	ep := fmt.Sprintf("/nations/%s", name)
	nation, err := utils.OAPIRequest[structs.NationInfo](ep, false)

	if err != nil { 
		return structs.NationInfo{}, err
	}
	
	return nation, nil
}