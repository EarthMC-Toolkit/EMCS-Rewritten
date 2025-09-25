package events

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/slashcommands"
	"emcsrw/utils"
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
		start := time.Now()

		staleTowns := append(make([]oapi.TownInfo, 0, len(towns)), towns...)

		var err error
		towns, err = PutFunc(db, "towns", func() ([]oapi.TownInfo, error) {
			return api.QueryAllTowns()
		})
		if err != nil {
			return
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

		PutFunc(db, "residentlist", func() (entities map[string]oapi.Entity, err error) {
			return residentlist, nil
		})

		nations, _ = PutFunc(db, "nations", func() ([]oapi.NationInfo, error) {
			res, _, _ := oapi.QueryConcurrent(lo.Keys(nationlist), oapi.QueryNations)
			return res, nil
		})
		//endregion

		//region ============ SPLIT RESIDENTS FROM TOWNLESS ============
		plist, err := oapi.QueryList(oapi.ENDPOINT_PLAYERS)
		if err != nil {
			return
		}

		townlesslist, _ = PutFunc(db, "townlesslist", func() (map[string]oapi.Entity, error) {
			entities := lo.FilterMap(plist, func(p oapi.Entity, _ int) (oapi.Entity, bool) {
				_, ok := residentlist[p.UUID]
				return p, !ok
			})

			// Convert slice to map using UUID as key.
			return lo.Associate(entities, func(p oapi.Entity) (string, oapi.Entity) {
				return p.UUID, p
			}), nil
		})
		//endregion

		fmt.Printf("\nTotal Players: %d, Residents: %d, Townless: %d", len(plist), len(residentlist), len(townlesslist))
		fmt.Printf("\nNations List: %d, Nations: %d", len(nationlist), len(nations))
		fmt.Printf("\n\nTook: %s\n\n", time.Since(start))

		TrySendLeftJoinedNotif(s, staleTowns)
		TrySendRuinedNotif(s, staleTowns)
	}, true, 30*time.Second)

	// Updating every min should be fine. doubt people care about having /vp and /serverinfo be realtime.
	scheduleTask(func() {
		info, err := PutFunc(db, "serverinfo", func() (oapi.ServerInfo, error) {
			return oapi.QueryServer()
		})
		if err != nil {
			return
		}

		TrySendVotePartyNotif(s, info.VoteParty)

		// remaining := info.VoteParty.NumRemaining
		// for _, threshold := range []int{1000, 500, 300, 150, 50} {
		// 	if remaining <= threshold && !voteNotified[threshold] {
		// 		msg := fmt.Sprintf("VoteParty has less than %d votes remaining!", threshold)
		// 		s.ChannelMessageSend("1420146203454083144", msg)
		// 		voteNotified[threshold] = true
		// 	}
		// }
	}, true, 60*time.Second)
}

func scheduleTask(task func(), runInitial bool, interval time.Duration) chan struct{} {
	if runInitial {
		task()
	}

	stop := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			task()
		}
	}()

	return stop
}

// Runs a task that returns a value, said value is then marshalled and stored in the given badger DB under the given key.
// If an error occurs during the task, the error is logged and returned, and the DB write will not occur.
func PutFunc[T any](mapDB *badger.DB, key string, task func() (T, error)) (T, error) {
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

	database.PutInsensitive(mapDB, key, data)
	log.Printf("put '%s' into db at %s\n", key, dbDir)

	return res, err
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
	staleTownsMap := lo.Associate(staleTowns, func(t oapi.TownInfo) (string, oapi.TownInfo) {
		return t.UUID, t
	})

	ruined := []oapi.TownInfo{}
	for _, t := range towns {
		wasRuined := staleTownsMap[t.UUID].Status.Ruined
		if t.Status.Ruined && !wasRuined {
			// Town has just been ruined.
			ruined = append(ruined, t)
		}
	}

	sort.Slice(ruined, func(i, j int) bool {
		return *ruined[i].Timestamps.RuinedAt > *ruined[j].Timestamps.RuinedAt
	})

	if len(ruined) > 0 {
		desc := lop.Map(ruined, func(t oapi.TownInfo, _ int) string {
			return fmt.Sprintf("`%s` fell into ruin <t:%d:R>", t.Name, *t.Timestamps.RuinedAt/1000)
		})

		s.ChannelMessageSendEmbed("1420855039357878469", &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Ruin Events [%d]", len(ruined)),
			Description: strings.Join(desc, "\n"),
		})
	}
}

// TODO: Increase sample size from 2 (last check and current) to like 5 with a sliding window for better rate/ETA accuracy.
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
