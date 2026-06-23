package tests

import (
	"emcsrw/pkg/api"
	"emcsrw/pkg/api/mapi"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/logutil"
	"errors"
	"testing"
	"time"

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
	t.Logf("Querying %d entities concurrently took %s", len(players), time.Since(start).Round(1*time.Second))

	logutil.LogValOrErr(t, players[0], errors.Join(errs...))
}

func TestGetVisiblePlayers(t *testing.T) {
	//t.SkipNow()

	res, err := mapi.GetVisiblePlayers()
	logutil.LogValOrErr(t, res, err)
}

func TestQueryVisiblePlayers(t *testing.T) {
	//t.SkipNow()

	players, _, err := api.QueryVisiblePlayers()
	logutil.LogValOrErr(t, players, err)
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
	logutil.LogValOrErr(t, towns[0], err)
}

func TestQueryTown(t *testing.T) {
	//t.SkipNow()

	town, err := api.QueryTown("Venice")
	logutil.LogValOrErr(t, town, err)
}

func TestQueryNation(t *testing.T) {
	//t.SkipNow()

	nation, err := api.QueryNation("Venice")
	logutil.LogValOrErr(t, nation, err)
}

func TestQueryPlayer(t *testing.T) {
	//t.SkipNow()

	player, err := api.QueryPlayer("Fruitloopins")
	logutil.LogValOrErr(t, player, err)
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

	t.Logf("Sent %d requests for %d players. Took %s", reqAmt, len(players), time.Since(start).Round(1*time.Second))

	// opNames := lo.FilterMap(players, func(p oapi.PlayerInfo, _ int) (string, bool) {
	// 	return p.Name, p.Status.IsOnline
	// })
	// slices.Sort(opNames)

	weekAgo := 7 * 24 * time.Hour
	monthAgo := 30 * 24 * time.Hour
	purgeAgo := 42 * 24 * time.Hour

	// cumulative stats
	var activeTownless7d, activeResidents7d int
	var activeTownless1m, activeResidents1m int
	var purgedTownless, purgedResidents int

	// retention in the last month
	var week1, week2, week3, week4 int
	var total, totalNew int
	var newActive7d, newActive30d int

	nostraRelease := time.UnixMilli(1776384000000)
	now := time.Now()
	for _, p := range players {
		if p.Timestamps.LastOnline == nil {
			continue // remove NPCs
		}
		total++

		registered := time.UnixMilli(int64(p.Timestamps.Registered))
		lo := time.UnixMilli(int64(*p.Timestamps.LastOnline))
		age := now.Sub(lo)

		if registered.After(nostraRelease) {
			totalNew++

			if age <= weekAgo {
				newActive7d++
			}
			if age <= monthAgo {
				newActive30d++
			}
		}

		if age <= weekAgo {
			if p.Status.HasTown {
				activeResidents7d++
			} else {
				activeTownless7d++
			}
		}
		if age <= monthAgo {
			if p.Status.HasTown {
				activeResidents1m++
			} else {
				activeTownless1m++
			}
		}
		if age >= purgeAgo {
			if p.Status.HasTown {
				purgedResidents++
			} else {
				purgedTownless++
			}
		}

		switch {
		case age <= weekAgo:
			week1++
		case age <= weekAgo*2:
			week2++
		case age <= weekAgo*3:
			week3++
		case age <= weekAgo*4:
			week4++
		}
	}

	t.Logf("\n")
	t.Logf("Total Players (excluding NPCS):				%d", total)
	t.Logf("Registered after Nostra release (Apr 17):	%d", totalNew)

	t.Logf("\n")
	t.Logf("Players online in the last 7 days (cumulative)")
	t.Logf("\tTownless:			%d", activeTownless7d)
	t.Logf("\tResidents: 		%d", activeResidents7d)
	t.Logf("\tCombined: 		%d\n", activeTownless7d+activeResidents7d)

	t.Logf("\n")
	t.Logf("Players online in the last 30 days (cumulative)")
	t.Logf("\tTownless: 		%d", activeTownless1m)
	t.Logf("\tResidents: 		%d", activeResidents1m)
	t.Logf("\tCombined: 		%d\n", activeTownless1m+activeResidents1m)

	t.Logf("\n")
	t.Logf("Purged players (>=42 days ago) (cumulative)")
	t.Logf("\tTownless: 		%d", purgedTownless)
	t.Logf("\tResidents: 		%d", purgedResidents)
	t.Logf("\tCombined: 		%d\n", purgedTownless+purgedResidents)

	retentionTotal := week1 + week2 + week3 + week4
	t.Logf("\n")
	t.Logf("Retention buckets (last 28 days)\n")
	t.Logf("\t22-28 days ago: 	%d (%.1f%%)", week4, 100*float64(week4)/float64(retentionTotal))
	t.Logf("\t15-21 days ago: 	%d (%.1f%%)", week3, 100*float64(week3)/float64(retentionTotal))
	t.Logf("\t8-14 days ago:  	%d (%.1f%%)", week2, 100*float64(week2)/float64(retentionTotal))
	t.Logf("\t0-7 days ago:		%d (%.1f%%)", week1, 100*float64(week1)/float64(retentionTotal))

	residentRetention := float64(activeResidents7d) / float64(activeResidents1m)
	townlessRetention := float64(activeTownless7d) / float64(activeTownless1m)

	t.Logf("\n")
	t.Logf("Resident weekly retention: %.1f%%", residentRetention*100)
	t.Logf("Townless weekly retention: %.1f%%", townlessRetention*100)

	newRetention7d := float64(newActive7d) / float64(totalNew) * 100
	newRetention30d := float64(newActive30d) / float64(totalNew) * 100

	t.Logf("\n")
	t.Logf("New player retention (7d): 	%.1f%%", newRetention7d)
	t.Logf("New player retention (30d): %.1f%%", newRetention30d)
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
