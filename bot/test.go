package bot

import (
	"emcs-rewritten/api/towns"
	"emcs-rewritten/utils"
	"fmt"
)

func Test() {
	town, err := towns.Get("venice")
	if err == nil {
		fmt.Print(utils.PrettyPrint(town))
		//fmt.Println(fmt.Sprintf("%s", fmt.Sprint(town.Stats.Balance)))
	}
}