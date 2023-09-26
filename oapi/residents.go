package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"fmt"
)

func Resident(identifier string) (structs.ResidentInfo, error) {
	endpoint := fmt.Sprintf("/residents/%s?", identifier)
	resident, err := utils.JsonRequest[structs.ResidentInfo](endpoint, true)

	if err != nil { 
		return structs.ResidentInfo{}, err
	}
	
	return resident, nil
}

func ConcurrentResidents(identifiers []string) ([]structs.ResidentInfo, []error) {
	var endpoints []string 
	for _, identifier := range identifiers {
		endpoints = append(endpoints, fmt.Sprintf("/residents/%s?", identifier))
	}

	return utils.ConcurrentJsonRequests[structs.ResidentInfo](endpoints, true)
}