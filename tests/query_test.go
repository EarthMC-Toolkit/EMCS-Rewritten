package tests

import (
	"emcsrw/pkg/api"
	"emcsrw/pkg/api/mapi"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/logutil"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/samber/lo/parallel"
)

type NationInactivity struct {
	Name      string
	Residents int
	AvgAge    time.Duration
}

type PlayerStats struct {
	Total    int
	TotalNew int

	ActiveTownless7d   int
	ActiveResidents7d  int
	ActiveTownless30d  int
	ActiveResidents30d int

	PurgedTownless  int
	PurgedResidents int

	Week1 int
	Week2 int
	Week3 int
	Week4 int

	NewActive7d  int
	NewActive30d int

	// TOP 10
	TopInactiveNations []NationInactivity
}

func (s PlayerStats) ActivityBucketTotal() int {
	return s.Week1 + s.Week2 + s.Week3 + s.Week4
}

func (s PlayerStats) Resident7dRate() float64 {
	if s.ActiveResidents30d == 0 {
		return 0
	}
	return float64(s.ActiveResidents7d) / float64(s.ActiveResidents30d)
}

func (s PlayerStats) Townless7dRate() float64 {
	if s.ActiveTownless30d == 0 {
		return 0
	}
	return float64(s.ActiveTownless7d) / float64(s.ActiveTownless30d)
}

func (s PlayerStats) NewPlayerActivity7d() float64 {
	if s.TotalNew == 0 {
		return 0
	}
	return float64(s.NewActive7d) / float64(s.TotalNew)
}

func (s PlayerStats) NewPlayerActivity30d() float64 {
	if s.TotalNew == 0 {
		return 0
	}
	return float64(s.NewActive30d) / float64(s.TotalNew)
}

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

	template := map[string]bool{"name": true, "nation": true, "timestamps": true, "status": true}

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
}

// To run this test with higher timeout, execute the following:
//
//	"go test -timeout 5m -run ^TestServerPlayerActivity$ emcsrw/tests -v -count=1"
func TestServerPlayerActivity(t *testing.T) {
	plist, _ := oapi.QueryList(oapi.ENDPOINT_PLAYERS).Execute()
	ids := parallel.Map(plist, func(p oapi.Entity, _ int) string {
		return p.UUID
	})
	if len(ids) < 1 {
		t.Fatal("invalid array len for player list")
	}

	t.Logf("Starting QueryConcurrent. Expect %d players. Tokens: %d", len(plist), len(oapi.Dispatcher.GetBucketTokens()))

	template := map[string]bool{"name": true, "nation": true, "timestamps": true, "status": true}

	start := time.Now()
	players, errs, reqAmt := oapi.QueryPlayers(ids...).WithTemplate(template).ExecuteConcurrent()
	if len(errs) > 0 {
		t.Fatal(errors.Join(errs...))
	}

	t.Logf("Sent %d requests for %d players. Took %s", reqAmt, len(players), time.Since(start).Round(1*time.Second))

	stats := CalculatePlayerStats(players)

	t.Logf("\n")
	t.Logf("Total Players (excluding NPCS):				%d", stats.Total)
	t.Logf("Registered after Nostra release (Apr 17):	%d", stats.TotalNew)

	t.Logf("\n")
	t.Logf("Players online in the last 7 days (cumulative)")
	t.Logf("\tTownless:			%d", stats.ActiveTownless7d)
	t.Logf("\tResidents: 		%d", stats.ActiveResidents7d)
	t.Logf("\tCombined: 		%d\n", stats.ActiveTownless7d+stats.ActiveResidents7d)

	t.Logf("\n")
	t.Logf("Players online in the last 30 days (cumulative)")
	t.Logf("\tTownless: 		%d", stats.ActiveTownless30d)
	t.Logf("\tResidents: 		%d", stats.ActiveResidents30d)
	t.Logf("\tCombined: 		%d\n", stats.ActiveTownless30d+stats.ActiveResidents30d)

	t.Logf("\n")
	t.Logf("Purged players (>=42 days ago) (cumulative)")
	t.Logf("\tTownless: 		%d", stats.PurgedTownless)
	t.Logf("\tResidents: 		%d", stats.PurgedResidents)
	t.Logf("\tCombined: 		%d\n", stats.PurgedTownless+stats.PurgedResidents)

	total := stats.ActivityBucketTotal()

	t.Logf("\n")
	t.Logf("Last activity distribution (last 28 days)")
	t.Logf("\t22-28 days ago:\t%d (%.1f%%)", stats.Week4, 100*float64(stats.Week4)/float64(total))
	t.Logf("\t15-21 days ago:\t%d (%.1f%%)", stats.Week3, 100*float64(stats.Week3)/float64(total))
	t.Logf("\t8-14 days ago:\t%d (%.1f%%)", stats.Week2, 100*float64(stats.Week2)/float64(total))
	t.Logf("\t0-7 days ago:\t%d (%.1f%%)", stats.Week1, 100*float64(stats.Week1)/float64(total))

	t.Logf("\n")
	t.Logf("Residents 7d activity rate: %.1f%%", stats.Resident7dRate()*100)
	t.Logf("Townless 7d activity rate: %.1f%%", stats.Townless7dRate()*100)

	t.Logf("\n")
	t.Logf("New player activity rate (7d): %.1f%%", stats.NewPlayerActivity7d()*100)
	t.Logf("New player activity rate (30d): %.1f%%", stats.NewPlayerActivity30d()*100)

	top := 10
	if len(stats.TopInactiveNations) > top {
		stats.TopInactiveNations = stats.TopInactiveNations[:top]
	}

	t.Logf("\n")
	t.Logf("Top inactive nations (avg last-seen age):")
	for i, n := range stats.TopInactiveNations {
		t.Logf("\t%d. %s - %.1f days (%d residents)", i+1, n.Name, n.AvgAge.Hours()/24, n.Residents)
	}
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

const weekAgo = 7 * 24 * time.Hour
const monthAgo = 30 * 24 * time.Hour
const purgeAgo = 42 * 24 * time.Hour

type nationAgg struct {
	totalAge time.Duration
	count    int
}

func CalculatePlayerStats(players []oapi.PlayerInfo) PlayerStats {
	nostraRelease := time.UnixMilli(1776384000000)
	now := time.Now()

	stats := PlayerStats{}
	nationMap := make(map[string]nationAgg)

	for _, p := range players {
		if p.Timestamps.LastOnline == nil {
			continue
		}

		stats.Total++

		registered := time.UnixMilli(int64(p.Timestamps.Registered))
		lastOnline := time.UnixMilli(int64(*p.Timestamps.LastOnline))
		age := now.Sub(lastOnline)

		if registered.After(nostraRelease) {
			stats.TotalNew++

			if age <= weekAgo {
				stats.NewActive7d++
			}
			if age <= monthAgo {
				stats.NewActive30d++
			}
		}

		if age <= monthAgo {
			if p.Status.HasTown {
				stats.ActiveResidents30d++
				if age <= weekAgo {
					stats.ActiveResidents7d++
				}
			} else {
				stats.ActiveTownless30d++
				if age <= weekAgo {
					stats.ActiveTownless7d++
				}
			}
		}
		if age >= purgeAgo {
			if p.Status.HasTown {
				stats.PurgedResidents++
			} else {
				stats.PurgedTownless++
			}
		}

		switch {
		case age <= weekAgo:
			stats.Week1++
		case age <= weekAgo*2:
			stats.Week2++
		case age <= weekAgo*3:
			stats.Week3++
		case age <= weekAgo*4:
			stats.Week4++
		}

		if p.Status.HasNation {
			nation := *p.Nation.Name
			n := nationMap[nation]
			n.totalAge += age
			n.count++
			nationMap[nation] = n
		}
	}

	// build top inactive nations
	var list []NationInactivity
	for name, n := range nationMap {
		if n.count < 20 { // filter tiny nations
			continue
		}

		list = append(list, NationInactivity{
			Name:      name,
			Residents: n.count,
			AvgAge:    n.totalAge / time.Duration(n.count),
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].AvgAge > list[j].AvgAge
	})

	stats.TopInactiveNations = list
	return stats
}
