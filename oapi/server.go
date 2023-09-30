package oapi

import (
	"emcsrw/oapi/structs"
	"emcsrw/utils"
	"errors"

	"github.com/samber/lo"
)

func ServerInfo() (structs.ServerInfo, error) {
	info, err := utils.OAPIRequest[structs.ServerInfo]("", false)

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

type BalanceTotals struct {
	Towns		*int
	Nations		*int
	Residents	*int
}

func ValidateOptType(value interface{}) (bool, error) {
    switch v := value.(type) {
    	case bool: return v, nil
    	default: return false, errors.New("Input value must be of type bool")
    }
}

type Entity struct {
	Name	string
}

func AllNames(ep string) ([]string, error) {
	res, err := utils.TKAPIRequest[[]Entity]("/residents")
	if err != nil {
		return nil, err
	}

	return lo.Map(res, func(e Entity, index int) string {
		return e.Name
	}), nil
}

// func WorldBalanceTotals(opts *BalanceOpts) (*BalanceTotals, error) {
// 	var err error
// 	var (
// 		includeTowns bool
// 		includeNations bool
// 		includeResidents bool
// 	)

// 	var (
// 		worldTownBal *int
// 		worldNationBal *int
// 		worldResidentBal *int
// 	)

//     includeTowns, err = ValidateOptType(opts.Towns)
// 	if err != nil {
// 		return nil, err
// 	}

// 	includeNations, err = ValidateOptType(opts.Nations)
// 	if err != nil {
// 		return nil, err
// 	}

// 	includeResidents, err = ValidateOptType(opts.Nations)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &BalanceTotals{
// 		Towns: worldTownBal,
// 		Nations: worldNationBal,
// 	}, nil
// }

type Stats interface {
	Bal() float32
}

func WorldBalance[T Stats](endpoint string) (int, error) {
	arr, err := utils.OAPIRequest[[]T](endpoint, false)
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