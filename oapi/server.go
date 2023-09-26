package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
)

func ServerInfo() (structs.ServerInfo, error) {
	info, err := utils.JsonRequest[structs.ServerInfo]("", false)

	if err != nil {
		return structs.ServerInfo{}, err
	}

	return info, nil
}

func ServerBalanceTotals() {
	
}