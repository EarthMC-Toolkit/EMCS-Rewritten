package residents

import (
	"emcs-rewritten/structs"
	"emcs-rewritten/utils"
	"fmt"
)

func Get(identifier string) (structs.ResidentInfo, error) {
	endpoint := fmt.Sprintf("/residents/%s?", identifier)
	resident, err := utils.JsonRequest[structs.ResidentInfo](endpoint, true)

	if err != nil { 
		return structs.ResidentInfo{}, err
	}
	
	return resident, nil
}