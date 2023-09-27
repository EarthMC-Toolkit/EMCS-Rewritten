package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"errors"

	"github.com/samber/lo"
)

func ServerInfo() (structs.ServerInfo, error) {
	info, err := utils.JsonRequest[structs.ServerInfo]("", false)

	if err != nil {
		return structs.ServerInfo{}, err
	}

	return info, nil
}

type BalanceOpts struct {
	Towns		interface{}
	Nations		interface{}
	Residents	interface{}
}

func ValidateOptType(value interface{}) (bool, error) {
    switch v := value.(type) {
    	case bool: return v, nil
    	default: return false, errors.New("Input value must be of type bool")
    }
}

func WorldBalanceTotals(opts *BalanceOpts) interface{} {
	// var err error
	// var (
	// 	includeTowns bool
	// 	includeNations bool
	// 	includeResidents bool
	// )

    // includeTowns, err = ValidateOptType(opts.Towns)
	// if err != nil {
	// 	return err
	// }

	// includeNations, err = ValidateOptType(opts.Nations)
	// if err != nil {
	// 	return err
	// }

	// includeResidents, err = ValidateOptType(opts.Residents)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func WorldNationBalance() (int, error) {
	nations, err := utils.JsonRequest[[]structs.NationInfo]("/nations", false)

	balances := lo.Map(nations, func(n structs.NationInfo, _ int) int {
		return int(n.Stats.Balance)
	})

	if err != nil { 
		return 0, err
	}
	
	sum := lo.Reduce(balances, func(agg int, item int, _ int) int {
		return agg + item
	}, 0)

	return sum, nil
}