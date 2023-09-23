package residents

import (
	"emcs-rewritten/structs"
	"emcs-rewritten/utils"
	"fmt"
)

func Get(name string) (structs.ResidentInfo, error) {
	resident, err := utils.JsonRequest[structs.ResidentInfo](fmt.Sprintf("/residents/%s?", name), true)

	if err != nil { 
		return structs.ResidentInfo{}, err
	}
	
	return resident, nil
}