package main

import (
	"emcsrw/api"
	"emcsrw/api/mapi"
	"emcsrw/api/oapi"
	"emcsrw/utils"
	"slices"
	"testing"
	"time"

	lop "github.com/samber/lo/parallel"
)

func TestQueryTown(t *testing.T) {
	//t.SkipNow()

	towns, err := oapi.QueryTowns("Venice")
	logVal(t, len(towns), err)
}

func TestQueryNation(t *testing.T) {
	//t.SkipNow()

	nations, err := oapi.QueryNations("Venice")
	logVal(t, len(nations), err)
}

func TestQueryPlayer(t *testing.T) {
	//t.SkipNow()

	players, err := oapi.QueryPlayers("Fruitloopins")
	logVal(t, len(players), err)
}

func TestQueryPlayersList(t *testing.T) {
	//t.SkipNow()

	plist, _ := oapi.QueryList(oapi.PLAYERS_ENDPOINT)
	names := lop.Map(plist, func(p oapi.Entity, _ int) string {
		return p.Name
	})

	start := time.Now()
	players, _, reqAmt := oapi.QueryPlayersConcurrent(names, 340)
	elapsed := time.Since(start)

	t.Logf("Sent %d requests for %d players. Took %s", reqAmt, len(players), elapsed)

	opNames := []string{}
	lop.ForEach(players, func(p oapi.PlayerInfo, _ int) {
		if p.Status.IsOnline {
			opNames = append(opNames, p.Name)
		}
	})

	slices.Sort(opNames)

	t.Logf("Total Online: %d\nNames: %v", len(opNames), opNames)
}

func TestGetOnlinePlayers(t *testing.T) {
	//t.SkipNow()

	res, err := mapi.GetOnlinePlayers()
	logVal(t, res, err)
}

func TestQueryOnlinePlayers(t *testing.T) {
	//t.SkipNow()

	players, err := api.QueryOnlinePlayers()
	logVal(t, len(players), err)
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
	players, errs, reqAmt := oapi.QueryPlayersConcurrent(names, 0)
	elapsed := time.Since(start)

	if len(errs) > 0 {
		t.Fatalf("Encountered %d errors during requests:", len(errs))
	}

	//logVal(t, players, nil)

	t.Logf("QueryPlayersConcurrent took %s. Sent %d requests containing %d players", elapsed, reqAmt, len(players))
}

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
