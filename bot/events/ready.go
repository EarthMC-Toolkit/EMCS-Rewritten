package events

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/slashcommands"
	"emcsrw/bot/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

// VoteParty notification tracking.
var vpThresholds = []int{500, 300, 150, 50}
var vpNotified = make(map[int]bool) // Key is each threshold, value is whether notified or not.
var vpLastRemaining int
var vpLastCheck time.Time

// TODO: We shouldn't really be registering commands here bc of the limit.
// Prefer a standalone script that registers via the REST API and run it when a command "definition" changes (such as description, options etc).
// To see changes after modifying the `Execute` func, just restart the client without running said script.
//
// https://discordjs.guide/creating-your-bot/command-deployment.html#command-registration.
func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Logged in as: %s\n\n", s.State.User.Username)
	slashcommands.SyncWithRemote(s)

	db, err := store.GetMapDB(common.ACTIVE_MAP)
	if err != nil {
		fmt.Printf("\n[OnReady]: wtf happened? error fetching db:\n%v", err)
		return
	}

	serverStore, _ := store.GetStore[oapi.ServerInfo](db, "server")

	// scheduleTask(func() {
	// 	PutFunc(db, "playerlist", func() ([]oapi.Entity, error) {
	// 		return oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	// 	})
	// }, true, 20*time.Second)

	scheduleTask(func() {
		fmt.Println()
		log.Println("[OnReady]: Running data update task...")

		start := time.Now()
		townList, staleTownList, err := UpdateData(db)
		elapsed := time.Since(start)

		fmt.Println()
		if err != nil {
			log.Println("[OnReady]: Failed data update task.")
			log.Println(err)
		}

		log.Println("[OnReady]: Completed data update task. Took: " + elapsed.String())

		staleTowns := lo.MapToSlice(staleTownList, func(_ string, t oapi.TownInfo) oapi.TownInfo {
			return t
		})

		//TrySendLeftJoinedNotif(s, *staleTowns)
		TrySendRuinedNotif(s, townList, staleTowns)
		TrySendFallenNotif(s, townList, staleTowns)
	}, true, 30*time.Second)

	// Updating every min should be fine. doubt people care about having /vp and /serverinfo be realtime.
	scheduleTask(func() {
		info, err := SetKeyFunc(serverStore, "info", func() (oapi.ServerInfo, error) {
			return oapi.QueryServer()
		})
		if err != nil {
			return
		}

		TrySendVotePartyNotif(s, info.VoteParty)

		if err := db.Flush(); err != nil {
			fmt.Println()
			log.Printf("error occurred flushing stores in db: %s", db.Dir())
		}
	}, true, 60*time.Second)
}

func scheduleTask(task func(), runInitial bool, interval time.Duration) {
	if runInitial {
		task()
	}

	ticker := time.NewTicker(interval)
	go func() {
		//defer ticker.Stop()
		for range ticker.C {
			task()
		}
	}()
}

// Runs a task that returns a value, said value is then marshalled and stored in the given badger DB under the given key.
// If an error occurs during the task, the error is logged and returned, and the DB write will not occur.
func OverwriteFunc[T any](store *store.Store[T], task func() (map[string]T, error)) (map[string]T, error) {
	v, err := task()
	if err != nil {
		log.Printf("error overwriting data in db at %s:\n%v", store.CleanPath(), err)
		return v, err
	}

	if len(v) < 1 {
		return nil, fmt.Errorf("error overwriting data in db at %s:\nretrieved value is empty", store.CleanPath())
	}

	store.Overwrite(v)
	//log.Printf("put '%s' into db at %s\n", key, dbDir)

	return v, err
}

func SetKeyFunc[T any](store *store.Store[T], key string, task func() (T, error)) (T, error) {
	res, err := task()
	if err != nil {
		log.Printf("error putting '%s' into db at %s:\n%v", key, store.CleanPath(), err)
		return res, err
	}

	store.SetKey(key, res)
	//log.Printf("put '%s' into db at %s\n", key, dbDir)

	return res, err
}

func UpdateData(db *store.MapDB) (map[string]oapi.TownInfo, map[string]oapi.TownInfo, error) {
	townStore, err := store.GetStore[oapi.TownInfo](db, "towns")
	if err != nil {
		return nil, nil, err
	}

	nationStore, err := store.GetStore[oapi.NationInfo](db, "nations")
	if err != nil {
		return nil, nil, err
	}

	entityStore, err := store.GetStore[oapi.EntityList](db, "entities")
	if err != nil {
		return nil, nil, err
	}

	staleTowns := townStore.Entries()
	fmt.Printf("DEBUG | Stale towns: %d", len(staleTowns))

	townList, err := OverwriteFunc(townStore, func() (map[string]oapi.TownInfo, error) {
		res, err := api.QueryAllTowns()
		if err != nil {
			return nil, err
		}

		return lo.SliceToMap(res, func(t oapi.TownInfo) (string, oapi.TownInfo) {
			return t.UUID, t
		}), nil
	})
	if err != nil {
		return townList, staleTowns, err
	}

	//region ============ GATHER DATA USING TOWNS ============
	residentList := make(oapi.EntityList)
	nlist := make(oapi.EntityList)
	for _, t := range townList {
		for _, r := range t.Residents {
			residentList[r.UUID] = r.Name
		}

		if t.Nation.UUID != nil {
			nlist[*t.Nation.UUID] = *t.Nation.Name
		}
	}

	SetKeyFunc(entityStore, "residentlist", func() (entities oapi.EntityList, err error) {
		return residentList, nil
	})

	nationList, _ := OverwriteFunc(nationStore, func() (map[string]oapi.NationInfo, error) {
		res, _, _ := oapi.QueryConcurrent(oapi.QueryNations, lo.Keys(nlist))
		return lo.SliceToMap(res, func(n oapi.NationInfo) (string, oapi.NationInfo) {
			return n.UUID, n
		}), nil
	})
	//endregion

	//region ============ SPLIT RESIDENTS FROM TOWNLESS ============
	players, err := oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	if err != nil {
		return staleTowns, townList, err
	}

	townlessList, _ := SetKeyFunc(entityStore, "townlesslist", func() (oapi.EntityList, error) {
		entities := lo.FilterMap(players, func(p oapi.Entity, _ int) (oapi.Entity, bool) {
			_, ok := residentList[p.UUID]
			return p, !ok
		})

		return lo.SliceToMap(entities, func(p oapi.Entity) (string, string) {
			return p.UUID, p.Name
		}), nil
	})
	//endregion

	fmt.Printf("\nDEBUG | Towns: %d, Nations: %d", len(townList), len(nationList))
	fmt.Printf("\nDEBUG | Total Players: %d, Residents: %d, Townless: %d", len(players), len(residentList), len(townlessList))

	return townList, staleTowns, err
}

// func CalcLeftJoined(staleTowns []oapi.TownInfo) (left, joined []string) {
// 	// build lookup maps
// 	staleResidents := make(map[string]oapi.TownInfo)
// 	for _, t := range staleTowns {
// 		for _, r := range t.Residents {
// 			staleResidents[r.UUID] = t
// 		}
// 	}

// 	currentResidents := make(map[string]oapi.TownInfo)
// 	for _, t := range towns {
// 		for _, r := range t.Residents {
// 			currentResidents[r.UUID] = t
// 		}
// 	}

// 	// who left
// 	for uuid, oldTown := range staleResidents {
// 		if _, ok := currentResidents[uuid]; !ok {
// 			name := townlesslist[uuid].Name
// 			nation := "No Nation"
// 			if oldTown.Nation != nil && oldTown.Nation.Name != nil {
// 				nation = *oldTown.Nation.Name
// 			}

// 			left = append(left, fmt.Sprintf("`%s` left %s (**%s**)", name, oldTown.Name, nation))
// 		}
// 	}

// 	// who joined
// 	for uuid, newTown := range currentResidents {
// 		if _, ok := staleResidents[uuid]; !ok {
// 			name := residentlist[uuid].Name
// 			nation := "No Nation"
// 			if newTown.Nation != nil && newTown.Nation.Name != nil {
// 				nation = *newTown.Nation.Name
// 			}

// 			joined = append(joined, fmt.Sprintf("`%s` joined %s (**%s**)", name, newTown.Name, nation))
// 		}
// 	}

// 	return
// }

// func TrySendLeftJoinedNotif(s *discordgo.Session, staleTowns []oapi.TownInfo) {
// 	left, joined := CalcLeftJoined(staleTowns)

// 	leftCount := len(left)
// 	joinedCount := len(joined)
// 	if (leftCount + joinedCount) > 0 {
// 		s.ChannelMessageSendEmbed("1420108251437207682", &discordgo.MessageEmbed{
// 			Title: "Player Flow | Town Join/Leave Events",
// 			Fields: []*discordgo.MessageEmbedField{
// 				{
// 					Name:   fmt.Sprintf("Became townless [%d]", leftCount),
// 					Value:  strings.Join(left, "\n"),
// 					Inline: true,
// 				},
// 				{
// 					Name:   fmt.Sprintf("Became a resident [%d]", joinedCount),
// 					Value:  strings.Join(joined, "\n"),
// 					Inline: true,
// 				},
// 			},
// 		})
// 	}
// }

func TrySendRuinedNotif(s *discordgo.Session, towns map[string]oapi.TownInfo, staleTowns []oapi.TownInfo) {
	staleRuined := lo.FilterSliceToMap(staleTowns, func(t oapi.TownInfo) (string, oapi.TownInfo, bool) {
		return t.UUID, t, t.Status.Ruined
	})

	ruined := lo.FilterMapToSlice(towns, func(_ string, t oapi.TownInfo) (oapi.TownInfo, bool) {
		_, wasRuined := staleRuined[t.UUID]
		return t, !wasRuined && t.Status.Ruined
	})

	sort.Slice(ruined, func(i, j int) bool {
		return *ruined[i].Timestamps.RuinedAt < *ruined[j].Timestamps.RuinedAt
	})

	count := len(ruined)
	if count > 0 {
		desc := lop.Map(ruined, func(t oapi.TownInfo, _ int) string {
			// nation := "No Nation"
			// if t.Nation.Name != nil {
			// 	nation = *t.Nation.Name
			// }

			ruinedTs := *t.Timestamps.RuinedAt
			deleteTs := time.UnixMilli(int64(ruinedTs)).Add(74 * time.Hour) // 72 UTC but EMC goes on UTC+2 (i think?)

			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			return fmt.Sprintf("`%s` fell into ruin <t:%d:R>. Located at %s\nDeletion in <t:%d:R>.", t.Name, ruinedTs/1000, locationLink, deleteTs.Unix())
		})

		s.ChannelMessageSendEmbed("1420855039357878469", &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Ruin Events [%d]", count),
			Description: strings.Join(desc, "\n"),
			Color:       discordutil.DARK_GOLD,
		})
	}
}

func TrySendFallenNotif(s *discordgo.Session, towns map[string]oapi.TownInfo, staleTowns []oapi.TownInfo) {
	tslice := lo.MapToSlice(towns, func(_ string, t oapi.TownInfo) oapi.TownInfo {
		return t
	})

	diff, _ := utils.DifferenceBy(staleTowns, tslice, func(t oapi.TownInfo) string {
		return t.UUID
	})

	count := len(diff)
	if count > 0 {
		desc := lop.Map(diff, func(t oapi.TownInfo, _ int) string {
			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			return fmt.Sprintf("`%s` was deleted. Located at %s", t.Name, locationLink)
		})

		s.ChannelMessageSendEmbed("1420855039357878469", &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Fall Events [%d]", count),
			Description: strings.Join(desc, "\n"),
			Color:       discordutil.RED,
		})
	}
}

// TODO: Increase sample size from 2 (last check and current) with a sliding window for better rate/ETA accuracy.
//
// For example, recording the last 15 minutes would be done using 15 samples, each sample at 60 second intervals.
func TrySendVotePartyNotif(s *discordgo.Session, vp oapi.ServerVoteParty) {
	remaining := vp.NumRemaining

	var rate float64
	var eta float64
	if !vpLastCheck.IsZero() {
		deltaVotes := vpLastRemaining - remaining
		deltaMinutes := time.Since(vpLastCheck).Minutes()
		if deltaVotes > 0 && deltaMinutes > 0 {
			rate = float64(deltaVotes) / deltaMinutes
			eta = float64(remaining) / rate
		}
	}

	for _, threshold := range vpThresholds {
		if remaining <= threshold && !vpNotified[threshold] {
			msg := fmt.Sprintf("VoteParty has less than `%d` votes remaining! Currently at `%d`.", threshold, remaining)
			if rate > 0 && eta > 0 {
				etaValue, etaUnit := utils.HumanizeDuration(eta)
				msg += fmt.Sprintf("\n\n:chart_with_upwards_trend: **Rate**: ~%.2f votes/min,\n:timer: **ETA**: %.1f %s", rate, etaValue, etaUnit)
			}

			s.ChannelMessageSend("1420146203454083144", msg)
			vpNotified[threshold] = true
		}
	}

	// reset when it finishes
	if remaining == 0 {
		vpNotified = make(map[int]bool)
	}

	vpLastRemaining = remaining
	vpLastCheck = time.Now()
}
