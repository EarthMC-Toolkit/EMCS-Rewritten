package tests

import (
	"emcsrw/api"
	"emcsrw/api/mapi"
	"emcsrw/api/oapi"
	"emcsrw/utils"
	"slices"
	"testing"
	"time"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

func isBetweenNowAndDuration(tsMilli *uint64, ago time.Duration) bool {
	if tsMilli == nil {
		return false
	}

	ts := time.UnixMilli(int64(*tsMilli))
	return ts.After(time.Now().Add(-ago)) && ts.Before(time.Now())
}

func TestGetOnlinePlayers(t *testing.T) {
	//t.SkipNow()

	res, err := mapi.GetVisiblePlayers()
	utils.CustomLog(t, res, err)
}

func TestQueryAllOnlinePlayers(t *testing.T) {
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

	towns, err := oapi.QueryTowns("Venice")
	utils.CustomLog(t, towns[0], err)
}

func TestQueryNation(t *testing.T) {
	//t.SkipNow()

	nations, err := oapi.QueryNations("Venice")
	utils.CustomLog(t, nations[0], err)
}

func TestQueryPlayer(t *testing.T) {
	//t.SkipNow()

	players, err := oapi.QueryPlayers("Fruitloopins")
	utils.CustomLog(t, players[0], err)
}

// To run this test with higher timeout, execute the following:
//
//	"go test -timeout 5m -run ^TestQueryPlayersList$ emcsrw/tests -v -count=1"
func TestQueryPlayersList(t *testing.T) {
	//t.SkipNow()

	plist, _ := oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	names := lop.Map(plist, func(p oapi.Entity, _ int) string {
		return p.Name
	})

	if len(names) < 1 {
		t.Fatal("invalid array len for player list")
	}

	t.Logf("Starting QueryConcurrent. Expect %d players. Tokens: %d", len(plist), len(oapi.Dispatcher.GetBucketTokens()))

	start := time.Now()
	players, _, reqAmt := oapi.QueryConcurrent(names, oapi.QueryPlayers)

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
	players, errs, reqAmt := oapi.QueryConcurrent(names, oapi.QueryPlayers)

	errCount := len(errs)
	if errCount > 0 {
		t.Fatalf("Encountered %d errors during requests:", errCount)
	}

	t.Logf("QueryPlayersConcurrent took %s. Sent %d requests containing %d players", time.Since(start), reqAmt, len(players))
}
