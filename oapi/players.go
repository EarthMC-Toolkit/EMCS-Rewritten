package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"fmt"
)

func Resident(identifier string) (structs.PlayerInfo, error) {
	endpoint := fmt.Sprintf("/players/%s", identifier)
	resident, err := utils.OAPIRequest[structs.PlayerInfo](endpoint)

	if err != nil {
		return structs.PlayerInfo{}, err
	}

	return resident, nil
}

func ConcurrentResidents(identifiers []string) ([]structs.PlayerInfo, []error) {
	endpoints := make([]string, len(identifiers))
	for _, identifier := range identifiers {
		endpoints = append(endpoints, fmt.Sprintf("/players/%s", identifier))
	}

	return utils.OAPIConcurrentRequest[structs.PlayerInfo](endpoints, true)
}
