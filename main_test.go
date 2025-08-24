package main

import (
	"emcsrw/mapi"
	"emcsrw/oapi"
	"emcsrw/utils"
	"testing"
	"time"

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

	// ops, err := mapi.GetOnlinePlayers()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// names := lop.Map(ops, func(op mapi.OnlinePlayer, _ int) string {
	// 	return op.Name
	// })

	nations, err := oapi.QueryNations("Venice")
	if err != nil {
		t.Fatal("error getting nation: Venice", err)
	}

	names := lop.Map(nations[0].Residents, func(p oapi.Entity, _ int) string {
		return p.Name
	})

	start := time.Now()
	players, errs, reqAmt := oapi.QueryPlayersConcurrent(names, oapi.PLAYERS_QUERY_LIMIT)
	elapsed := time.Since(start)

	if len(errs) > 0 {
		t.Fatalf("Encountered %d errors during requests:", len(errs))
	}

	//logVal(t, players, nil)

	t.Logf("QueryPlayersConcurrent took %s. Sent %d requests containing %d players", elapsed, reqAmt, len(players))
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

type Loggable interface {
	Log(args ...any)
}

func logVal(t Loggable, v any, err error) {
	if err != nil {
		t.Log(err)
	} else {
		t.Log(utils.Prettify(v))
	}
}
