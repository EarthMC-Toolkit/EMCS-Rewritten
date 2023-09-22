package bot

import (
	"emcs-rewritten/classes/towns"
	"fmt"
)

func Test() {
	town, err := towns.Get("venice")
	if err == nil {
		//fmt.Print(prettyPrint(town))
		fmt.Println(fmt.Sprintf("%s", fmt.Sprint(town.Stats.Balance)))
	}
}