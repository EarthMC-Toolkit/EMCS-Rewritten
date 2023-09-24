package residents

import (
	"emcs-rewritten/utils"
	"emcs-rewritten/structs"
	"fmt"
)

func Get(identifier string) (structs.ResidentInfo, error) {
	resident, err := utils.JsonRequest[structs.ResidentInfo](fmt.Sprintf("/residents/%s?", identifier), true)

	if err != nil { 
		return structs.ResidentInfo{}, err
	}
	
	return resident, nil
}