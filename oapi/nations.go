package oapi

import (
	"emcsrw/utils"
	"emcsrw/oapi/structs"
	"fmt"
)

func Nation(identifier string) (structs.NationInfo, error) {
	ep := fmt.Sprintf("/nations/%s", identifier)
	nation, err := utils.OAPIRequest[structs.NationInfo](ep, false)

	if err != nil { 
		return structs.NationInfo{}, err
	}
	
	return nation, nil
}

func ConcurrentNations(identifiers []string) ([]structs.NationInfo, []error) {
	var endpoints []string 
	for _, identifier := range identifiers {
		endpoints = append(endpoints, fmt.Sprintf("/nations/%s?", identifier))
	}

	fmt.Println(endpoints)
	return utils.OAPIConcurrentRequest[structs.NationInfo](endpoints, true)
}