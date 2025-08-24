package main

import (
	"emcsrw/mapi"
	"emcsrw/oapi"
	"emcsrw/utils"
	"testing"

	lop "github.com/samber/lo/parallel"
)

func TestQueryTown(t *testing.T) {
	//t.SkipNow()

	town, err := oapi.QueryTowns("Venice")
	logVal(t, town, err)
}

func TestQueryNation(t *testing.T) {
	//t.SkipNow()

	nation, err := oapi.QueryNations("Venice")
	logVal(t, nation, err)
}

func TestQueryPlayer(t *testing.T) {
	//t.SkipNow()

	res, err := oapi.QueryPlayers("Fruitloopins")
	logVal(t, res, err)
}

func TestGetOnlinePlayers(t *testing.T) {
	//t.SkipNow()

	res, err := mapi.GetOnlinePlayers()
	logVal(t, res, err)
}

func TestQueryPlayersConcurrent(t *testing.T) {
	//t.SkipNow()

	ops, err := mapi.GetOnlinePlayers()
	if err != nil {
		t.Fatal(err)
	}

	opNames := lop.Map(ops, func(op mapi.OnlinePlayer, _ int) string {
		return op.Name
	})

	res, _ := oapi.QueryPlayersConcurrent(opNames, 100)
	logVal(t, res, nil)
}

// func TestConcurrentResidents(t *testing.T) {
// 	//t.SkipNow()

// 	residents, _ := oapi.ConcurrentResidents([]string{"Owen3H", "Fruitloopins"})
// 	logVal(t, residents, nil)
// }

func TestAlphanumeric(t *testing.T) {
	var actual bool

	actual = utils.ContainsNonAlphanumeric("<blue> owen")
	if actual != true {
		t.Errorf("Expected '%v' but got '%v'", true, actual)
	}

	actual = utils.ContainsNonAlphanumeric("owen3h")
	if actual != false {
		t.Errorf("Expected '%v' but got '%v'", false, actual)
	}
}

// func TestWorldNationBal(t *testing.T) {
// 	bal := oapi.WorldNationBalance()
// 	logVal(t, bal, nil)
// }

// func TestWorldBalances(t *testing.T) {
// 	worldBals, _ := oapi.WorldBalanceTotals(&oapi.BalanceOpts{
// 		Towns: true,
// 		Nations: true,
// 	})

// 	logVal(t, worldBals, nil)
// }

// func TestResidentNames(t *testing.T) {
// 	names, err := oapi.AllNames("/residents")
// 	//residents, _ := oapi.ConcurrentResidents(names)
// 	logVal(t, names, err)
// }

func logVal(t *testing.T, v any, err error) {
	if err != nil {
		t.Log(err)
	} else {
		t.Log(utils.Prettify(v))
	}
}
