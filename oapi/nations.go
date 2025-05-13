package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"fmt"
)

func Nation(identifier string) (structs.NationInfo, error) {
	ep := fmt.Sprintf("/nations/%s", identifier)
	nation, err := utils.OAPIRequest[structs.NationInfo](ep)

	if err != nil {
		return structs.NationInfo{}, err
	}

	return nation, nil
}

func ConcurrentNations(identifiers []string) ([]structs.NationInfo, []error) {
	var endpoints []string
	for _, identifier := range identifiers {
		endpoints = append(endpoints, fmt.Sprintf("/nations/%s", identifier))
	}

	return utils.OAPIConcurrentRequest[structs.NationInfo](endpoints, true)
}
