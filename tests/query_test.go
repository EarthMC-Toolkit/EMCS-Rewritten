package tests

import (
	"emcsrw/api"
	"emcsrw/api/mapi"
	"emcsrw/api/oapi"
	"emcsrw/utils"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

func isBetweenNowAndDuration(tsMilli *uint64, ago time.Duration) bool {
	if tsMilli == nil {
		return false
	}

	ts := time.UnixMilli(int64(*tsMilli))
	return ts.After(time.Now().Add(-ago)) && ts.Before(time.Now())
}

func TestExecuteConcurrent(t *testing.T) {
	nation, _ := api.QueryNation("Cascadia") // ~2k residents
	ids := parallel.Map(nation.Residents, func(e oapi.Entity, _ int) string {
		return e.UUID
	})

	start := time.Now()
	players, errs, _ := oapi.QueryPlayers(ids...).ExecuteConcurrent()
	t.Logf("Querying %d entities concurrently took %s", len(players), time.Since(start))

	utils.CustomLog(t, players[0], errors.Join(errs...))
}

func TestGetVisiblePlayers(t *testing.T) {
	//t.SkipNow()

	res, err := mapi.GetVisiblePlayers()
	utils.CustomLog(t, res, err)
}

func TestQueryVisiblePlayers(t *testing.T) {
	//t.SkipNow()

	players, err := api.QueryVisiblePlayers()
	utils.CustomLog(t, len(players), err)
}

func TestQueryAllTowns(t *testing.T) {
	//t.SkipNow()

	towns, err := api.QueryAllTowns()
	if err != nil {
		t.Fatal("error querying all towns", err)
	}

	count := len(towns)
	if count < 1 {
		t.Fatal("error querying all towns. no towns retreived")
	}

	t.Log(count)
	utils.CustomLog(t, towns[0], err)
}

func TestQueryTown(t *testing.T) {
	//t.SkipNow()

	town, err := api.QueryTown("Venice")
	utils.CustomLog(t, town, err)
}

func TestQueryNation(t *testing.T) {
	//t.SkipNow()

	nation, err := api.QueryNation("Venice")
	utils.CustomLog(t, nation, err)
}

func TestQueryPlayer(t *testing.T) {
	//t.SkipNow()

	player, err := api.QueryPlayer("Fruitloopins")
	utils.CustomLog(t, player, err)
}

// To run this test with higher timeout, execute the following:
//
//	"go test -timeout 5m -run ^TestQueryPlayersList$ emcsrw/tests -v -count=1"
func TestQueryPlayersList(t *testing.T) {
	//t.SkipNow()

	plist, _ := oapi.QueryList(oapi.ENDPOINT_PLAYERS).Execute()
	ids := parallel.Map(plist, func(p oapi.Entity, _ int) string {
		return p.UUID
	})
	if len(ids) < 1 {
		t.Fatal("invalid array len for player list")
	}

	t.Logf("Starting QueryConcurrent. Expect %d players. Tokens: %d", len(plist), len(oapi.Dispatcher.GetBucketTokens()))

	template := map[string]bool{"name": true, "timestamps": true, "status": true}

	start := time.Now()
	players, errs, reqAmt := oapi.QueryPlayers(ids...).WithTemplate(template).ExecuteConcurrent()
	if len(errs) > 0 {
		t.Fatal(errors.Join(errs...))
	}

	t.Logf("Sent %d requests for %d players. Took %s", reqAmt, len(players), time.Since(start))

	opNames := lo.FilterMap(players, func(p oapi.PlayerInfo, _ int) (string, bool) {
		return p.Name, p.Status.IsOnline
	})
	slices.Sort(opNames)

	opNames72h := lo.FilterMap(players, func(p oapi.PlayerInfo, _ int) (string, bool) {
		return p.Name, isBetweenNowAndDuration(p.Timestamps.LastOnline, 72*time.Hour)
	})
	slices.Sort(opNames72h)

	t.Logf("Currently Online: %d\nNames: %v", len(opNames), opNames)
	t.Logf("Online Last 72h: %d\nNames: %v", len(opNames72h), opNames72h)
}

// func TestQueryPlayersConcurrent(t *testing.T) {
// 	//t.SkipNow()

// 	ops, err := mapi.GetVisiblePlayers()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	ids := lop.Map(ops, func(p mapi.MapPlayer, _ int) string {
// 		return p.UUID
// 	})
// 	if len(ids) < 1 {
// 		t.Fatal("invalid array len for player list")
// 	}

// 	start := time.Now()

// 	players, errs, reqAmt := oapi.QueryPlayers(ids...).ExecuteConcurrent()
// 	errCount := len(errs)
// 	if errCount > 0 {
// 		t.Fatalf("Encountered %d errors during requests:", errCount)
// 	}

// 	t.Logf("QueryPlayersConcurrent took %s. Sent %d requests containing %d players", time.Since(start), reqAmt, len(players))
// }
