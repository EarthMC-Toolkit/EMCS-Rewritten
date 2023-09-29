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
	//Residents	interface{}
}

type BalanceTotals struct {
	Towns		*int
	Nations		*int
	//Residents	interface{}
}

func ValidateOptType(value interface{}) (bool, error) {
    switch v := value.(type) {
    	case bool: return v, nil
    	default: return false, errors.New("Input value must be of type bool")
    }
}

func WorldBalanceTotals(opts *BalanceOpts) *BalanceTotals {
	var err error
	var (
		includeTowns bool
		includeNations bool
		//includeResidents bool
	)

    includeTowns, err = ValidateOptType(opts.Towns)
	if err != nil {
		return nil
	}

	includeNations, err = ValidateOptType(opts.Nations)
	if err != nil {
		return nil
	}

	// includeResidents, err = ValidateOptType(opts.Residents)
	// if err != nil {
	// 	return err
	// }

	var worldTownBal *int
	if includeTowns == true {
		val, _ := WorldBalance[structs.TownInfo]("/towns/")
		if val != 0 {
			worldTownBal = &val
		}
	}

	var worldNationBal *int
	if includeNations == true {
		val, _ := WorldBalance[structs.NationInfo]("/nations")
		if val != 0 {
			worldNationBal = &val
		}
	}

	return &BalanceTotals{
		Towns: worldTownBal,
		Nations: worldNationBal,
	}
}

type Stats interface {
	Bal() float32
}

func WorldBalance[T Stats](endpoint string) (int, error) {
	arr, err := utils.JsonRequest[[]T](endpoint, false)
	balances := lo.Map(arr, func(t T, _ int) int {
		return int(t.Bal())
	})

	if err != nil { return 0, err }
	return CalcTotal(balances), nil
}

func CalcTotal(balances []int) int {
	reducer := func(agg int, item int, _ int) int { return agg + item }
	return lo.Reduce(balances, reducer, 0)
}