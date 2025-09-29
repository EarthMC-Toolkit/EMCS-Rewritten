package events

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/slashcommands"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

var nations []oapi.NationInfo
var towns []oapi.TownInfo
var townlesslist map[string]oapi.Entity
var residentlist map[string]oapi.Entity

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

	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	if db == nil {
		fmt.Println("[OnReady]: wtf happened? db is nil")
		return
	}

	// scheduleTask(func() {
	// 	PutFunc(db, "playerlist", func() ([]oapi.Entity, error) {
	// 		return oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	// 	})
	// }, true, 20*time.Second)

	scheduleTask(func() {
		log.Println("[OnReady]: Running main data update task...")
		start := time.Now()
		staleTowns := UpdateData(db)
		log.Printf("\n\nTask complete. Took: %s\n\n", time.Since(start))

		TrySendLeftJoinedNotif(s, *staleTowns)
		TrySendRuinedNotif(s, *staleTowns)
		TrySendFallenNotif(s, *staleTowns)
	}, true, 30*time.Second)

	// Updating every min should be fine. doubt people care about having /vp and /serverinfo be realtime.
	scheduleTask(func() {
		info, err := PutFunc(db, "serverinfo", 120*time.Second, func() (oapi.ServerInfo, error) {
			return oapi.QueryServer()
		})
		if err != nil {
			return
		}

		TrySendVotePartyNotif(s, info.VoteParty)
	}, true, 60*time.Second)

	// Clean up stale DB entries
	scheduleDatabaseGC(db, 5*time.Minute)
}

func scheduleDatabaseGC(db *badger.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
	retry:
		err := db.RunValueLogGC(0.1)
		switch err {
		case nil:
			goto retry
		case badger.ErrNoRewrite:
		default:
			log.Printf("badger GC error: %v", err)
		}
	}
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
func PutFunc[T any](mapDB *badger.DB, key string, ttl time.Duration, task func() (T, error)) (T, error) {
	dbDir := mapDB.Opts().Dir

	res, err := task()
	if err != nil {
		log.Printf("error putting '%s' into db at %s:\n%v", key, dbDir, err)
		return res, err
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("error putting '%s' into db at %s:\n%v", key, dbDir, err)
		return res, err
	}

	database.PutInsensitiveTTL(mapDB, key, data, ttl)
	//log.Printf("put '%s' into db at %s\n", key, dbDir)

	return res, err
}

func UpdateData(db *badger.DB) *[]oapi.TownInfo {
	staleTowns, err := database.GetInsensitive[[]oapi.TownInfo](db, "towns")
	if err != nil {
		staleTowns = &[]oapi.TownInfo{}
	}

	towns, err = PutFunc(db, "towns", 60*time.Second, func() ([]oapi.TownInfo, error) {
		return api.QueryAllTowns()
	})
	if err != nil {
		return staleTowns
	}

	//region ============ GATHER DATA USING TOWNS ============
	residentlist = make(map[string]oapi.Entity)
	nationlist := make(map[string]oapi.Entity)
	for _, t := range towns {
		for _, r := range t.Residents {
			residentlist[r.UUID] = r
		}

		if t.Nation.UUID != nil {
			nationlist[*t.Nation.UUID] = oapi.Entity{
				Name: *t.Nation.Name,
				UUID: *t.Nation.UUID,
			}
		}
	}

	PutFunc(db, "residentlist", 60*time.Second, func() (entities map[string]oapi.Entity, err error) {
		return residentlist, nil
	})

	nations, _ = PutFunc(db, "nations", 60*time.Second, func() ([]oapi.NationInfo, error) {
		res, _, _ := oapi.QueryConcurrent(lo.Keys(nationlist), oapi.QueryNations)
		return res, nil
	})
	//endregion

	//region ============ SPLIT RESIDENTS FROM TOWNLESS ============
	playerlist, err := oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	if err != nil {
		return staleTowns
	}

	townlesslist, _ = PutFunc(db, "townlesslist", 60*time.Second, func() (map[string]oapi.Entity, error) {
		entities := lo.FilterMap(playerlist, func(p oapi.Entity, _ int) (oapi.Entity, bool) {
			_, ok := residentlist[p.UUID]
			return p, !ok
		})

		// Convert slice to map using UUID as key.
		return lo.Associate(entities, func(p oapi.Entity) (string, oapi.Entity) {
			return p.UUID, p
		}), nil
	})
	//endregion

	fmt.Printf("\nDEBUG | Total Players: %d, Residents: %d, Townless: %d", len(playerlist), len(residentlist), len(townlesslist))
	fmt.Printf("\nDEBUG | Nations List: %d, Nations: %d, Towns: %d", len(nationlist), len(nations), len(towns))

	return staleTowns
}

func CalcLeftJoined(staleTowns []oapi.TownInfo) (left, joined []string) {
	// build lookup maps
	staleResidents := make(map[string]oapi.TownInfo)
	for _, t := range staleTowns {
		for _, r := range t.Residents {
			staleResidents[r.UUID] = t
		}
	}

	currentResidents := make(map[string]oapi.TownInfo)
	for _, t := range towns {
		for _, r := range t.Residents {
			currentResidents[r.UUID] = t
		}
	}

	// who left
	for uuid, oldTown := range staleResidents {
		if _, ok := currentResidents[uuid]; !ok {
			name := townlesslist[uuid].Name
			nation := "No Nation"
			if oldTown.Nation != nil && oldTown.Nation.Name != nil {
				nation = *oldTown.Nation.Name
			}

			left = append(left, fmt.Sprintf("`%s` left %s (**%s**)", name, oldTown.Name, nation))
		}
	}

	// who joined
	for uuid, newTown := range currentResidents {
		if _, ok := staleResidents[uuid]; !ok {
			name := residentlist[uuid].Name
			nation := "No Nation"
			if newTown.Nation != nil && newTown.Nation.Name != nil {
				nation = *newTown.Nation.Name
			}

			joined = append(joined, fmt.Sprintf("`%s` joined %s (**%s**)", name, newTown.Name, nation))
		}
	}

	return
}

func TrySendLeftJoinedNotif(s *discordgo.Session, staleTowns []oapi.TownInfo) {
	left, joined := CalcLeftJoined(staleTowns)

	leftCount := len(left)
	joinedCount := len(joined)
	if (leftCount + joinedCount) > 0 {
		s.ChannelMessageSendEmbed("1420108251437207682", &discordgo.MessageEmbed{
			Title: "Player Flow | Town Join/Leave Events",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   fmt.Sprintf("Became townless [%d]", leftCount),
					Value:  strings.Join(left, "\n"),
					Inline: true,
				},
				{
					Name:   fmt.Sprintf("Became a resident [%d]", joinedCount),
					Value:  strings.Join(joined, "\n"),
					Inline: true,
				},
			},
		})
	}
}

func TrySendRuinedNotif(s *discordgo.Session, staleTowns []oapi.TownInfo) {
	staleRuined := lo.FilterSliceToMap(staleTowns, func(t oapi.TownInfo) (string, oapi.TownInfo, bool) {
		return t.UUID, t, t.Status.Ruined
	})

	ruined := lo.FilterMap(towns, func(t oapi.TownInfo, _ int) (oapi.TownInfo, bool) {
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

			return fmt.Sprintf("`%s` fell into ruin <t:%d:R>. Deletion in <t:%d:R>.", t.Name, ruinedTs/1000, deleteTs.Unix())
		})

		s.ChannelMessageSendEmbed("1420855039357878469", &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Ruin Events [%d]", count),
			Description: strings.Join(desc, "\n"),
			Color:       discordutil.DARK_GOLD,
		})
	}
}

func TrySendFallenNotif(s *discordgo.Session, staleTowns []oapi.TownInfo) {
	diff, _ := utils.DifferenceBy(staleTowns, towns, func(t oapi.TownInfo) string {
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
