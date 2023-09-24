package towns

import (
	"emcs-rewritten/utils"
	"emcs-rewritten/structs"
	"fmt"
)

func Get(name string) (structs.TownInfo, error) {
	town, err := utils.JsonRequest[structs.TownInfo](fmt.Sprintf("/towns/%s?", name), false)

	if err != nil { 
		return structs.TownInfo{}, err
	}
	
	return town, nil
}