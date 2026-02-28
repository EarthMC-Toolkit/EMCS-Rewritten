package events

import (
	"cmp"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

// max amount of messages to fetch from news channel during its scheduled task.
// should be enough to cover at least a few days/weeks of news depending on activity.
const NEWS_CHANNEL_MAX_MSGS = 100
const NEWS_CHANNEL_BATCH_MSGS = 5

const NEWS_CHANNEL_ID = "970962878486183958"
const TFLOW_CHANNEL_ID = "1420855039357878469"
const VP_CHANNEL_ID = "1420146203454083144"

// VoteParty notification tracking.
var vpThresholds = [...]int{500, 300, 150, 50}
var vpNotified = make(map[int]bool) // Key is each threshold, value is whether notified or not.
var vpLastRemaining int
var vpLastCheck time.Time

var readyOnce sync.Once // This prevents running tasks more than once if OnReady is called multiple times.

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as: %s\n", s.State.User.Username)

	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		fmt.Printf("\n[OnReady]: wtf happened? error fetching db:\n%v", err)
		return
	}

	readyOnce.Do(func() {
		startTasks(s, mdb)
	})
}

func startTasks(s *discordgo.Session, mdb *database.Database) {
	scheduleTask(func() {
		fmt.Println()
		log.Println("[OnReady]: Running data update task...")

		start := time.Now()
		townList, staleTownList, townless, residents, err := UpdateData(mdb)

		fmt.Println() // use \n without log.Printf messing up date/time
		if err != nil {
			log.Printf("[OnReady]: Failed data update task.\n%s\n", err)
		} else {
			elapsed := time.Since(start)
			log.Printf("[OnReady]: Completed data update task. Took: %s\n", elapsed.String())
		}

		towns := lo.MapToSlice(townList, func(_ string, t oapi.TownInfo) oapi.TownInfo { return t })
		staleTowns := lo.MapToSlice(staleTownList, func(_ string, t oapi.TownInfo) oapi.TownInfo { return t })

		TrySendLeftJoinedNotif(s, towns, staleTowns, townless, residents)
		TrySendRuinedNotif(s, townList, staleTowns)
		TrySendFallenNotif(s, townList, staleTowns)
	}, true, 30*time.Second)

	serverStore, err := database.GetStore(mdb, database.SERVER_STORE)
	if err != nil {
		fmt.Printf("\nERROR | cannot schedule serverinfo task:\n\t%s", err)
		return
	}
	scheduleTask(func() {
		if info, err := SetKeyFunc(serverStore, "info", func() (oapi.ServerInfo, error) {
			return oapi.QueryServer()
		}); err == nil {
			TrySendVotePartyNotif(s, info.VoteParty)
			if err := serverStore.WriteSnapshot(); err != nil {
				fmt.Printf("\nERROR | server store failed to write snapshot:\n\t%s", err)
			}
		}
	}, true, 1*time.Minute) // Updating every min should be fine. doubt people care about having /vp and /serverinfo be realtime.

	newsStore, err := database.GetStore(mdb, database.NEWS_STORE)
	if err != nil {
		fmt.Printf("\nERROR | cannot schedule news task:\n\t%s", err)
		return
	}
	scheduleTask(func() {
		if _, err := OverwriteFunc(newsStore, func() (map[string]database.NewsEntry, error) {
			newsMsgs, err := discordutil.FetchMessages(s, NEWS_CHANNEL_ID, NEWS_CHANNEL_MAX_MSGS, NEWS_CHANNEL_BATCH_MSGS)
			if err != nil {
				return nil, err
			}

			return database.MessagesToNewsEntries(newsMsgs), nil
		}); err == nil {
			if err := newsStore.WriteSnapshot(); err != nil {
				fmt.Printf("\nERROR | news store failed to write snapshot:\n\t%s", err)
			}
		}
	}, true, 2*time.Minute)
}

func scheduleTask(task func(), runInitial bool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		if runInitial {
			go task()
		}

		// defer ticker.Stop()
		for range ticker.C {
			task()
		}
	}()
}

// Runs task whos returned value is used to overwrite the data within store.
//
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

	store.Set(key, res)
	//log.Printf("put '%s' into db at %s\n", key, dbDir)

	return res, err
}

func UpdateData(mdb *database.Database) (
	towns map[string]oapi.TownInfo, staleTowns map[string]oapi.TownInfo,
	townless, residents oapi.EntityList, err error,
) {
	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	entityStore, err := database.GetStore(mdb, database.ENTITIES_STORE)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	playerStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	staleTowns = townStore.Entries()
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
		return townList, staleTowns, nil, nil, err
	}

	//region ============ GATHER DATA USING TOWNS ============
	residentList, nlist := make(oapi.EntityList), make(oapi.EntityList)
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
		return townList, staleTowns, nil, residentList, err
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

	// Use pointers so the town struct isn't copied every time.
	// This should help with mem usage when we use it in building basic player map.
	playerTownLookup := make(map[string]*oapi.TownInfo, len(residentList))
	for _, town := range townList {
		for _, r := range town.Residents {
			t := town
			playerTownLookup[r.UUID] = &t
		}
	}

	OverwriteFunc(playerStore, func() (map[string]database.BasicPlayer, error) {
		playersMap := make(map[string]database.BasicPlayer)
		for uuid, name := range townlessList {
			playersMap[uuid] = database.NewBasicPlayerEntity(uuid, name)
		}
		for uuid, name := range residentList {
			bp := database.NewBasicPlayerEntity(uuid, name)

			// Get player town by their UUID. While the town should always exist,
			// this prevents a potential panic and keeps them townless.
			if t, ok := playerTownLookup[uuid]; ok {
				bp.Town = &t.Entity
				if t.Nation.UUID != nil {
					bp.Nation = &oapi.Entity{Name: *t.Nation.Name, UUID: *t.Nation.UUID}
				}
			}

			playersMap[uuid] = bp
		}

		return playersMap, nil
	})

	fmt.Printf("\nDEBUG | Towns: %d, Nations: %d", len(townList), len(nationList))
	fmt.Printf("\nDEBUG | Total Players: %d, Residents: %d, Townless: %d", len(players), len(residentList), len(townlessList))

	return townList, staleTowns, townlessList, residentList, err
}

// #region Channel notifs
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
			content := fmt.Sprintf("VoteParty has less than `%d` votes remaining! Currently at `%d`.", threshold, remaining)
			if rate > 0 && eta > 0 {
				etaValue, etaUnit := utils.HumanizeDuration(eta)
				content += fmt.Sprintf(
					"\n\n:chart_with_upwards_trend: **Rate**: ~%.2f votes/min,\n:timer: **ETA**: %.1f %s",
					rate, etaValue, etaUnit,
				)
			}

			// Send notif to toolkit #voteparty channel and publish to followers.
			if msg, err := s.ChannelMessageSend(VP_CHANNEL_ID, content); err == nil {
				s.ChannelMessageCrosspost(VP_CHANNEL_ID, msg.ID)
			}

			vpNotified[threshold] = true
		}
	}

	// finished, reset notification statuses by emptying map.
	if remaining == 0 {
		vpNotified = make(map[int]bool)
	}

	vpLastRemaining = remaining
	vpLastCheck = time.Now()
}

func TrySendRuinedNotif(s *discordgo.Session, towns map[string]oapi.TownInfo, staleTowns []oapi.TownInfo) {
	staleRuined := lo.FilterSliceToMap(staleTowns, func(t oapi.TownInfo) (string, oapi.TownInfo, bool) {
		return t.UUID, t, t.Status.Ruined
	})

	ruined := lo.FilterMapToSlice(towns, func(_ string, t oapi.TownInfo) (oapi.TownInfo, bool) {
		_, wasRuined := staleRuined[t.UUID]
		return t, !wasRuined && t.Status.Ruined
	})

	// Sort by oldest RuinedAt (aka least amount of time until deletion)
	slices.SortFunc(ruined, func(a, b oapi.TownInfo) int {
		return cmp.Compare(*a.Timestamps.RuinedAt, *b.Timestamps.RuinedAt)
	})

	count := len(ruined)
	if count > 0 {
		desc := parallel.Map(ruined, func(t oapi.TownInfo, _ int) string {
			chunks := utils.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := utils.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

			ruinedTs := *t.Timestamps.RuinedAt
			ruinedTime := time.UnixMilli(int64(*t.Timestamps.RuinedAt))
			after72h := ruinedTime.Add(72 * time.Hour)

			nextNewDay := time.Date(after72h.Year(), after72h.Month(), after72h.Day(), 11, 0, 0, 0, time.UTC)
			if !nextNewDay.After(after72h) {
				nextNewDay = nextNewDay.Add(24 * time.Hour)
			}

			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf(
				"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)",
				spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
			)

			return fmt.Sprintf(
				"`%s` fell into ruin <t:%d:R> at %s. %sG %s\nDeletion on `%s` (<t:%d:R>).",
				t.Name, ruinedTs/1000, locationLink, balance, chunks, utils.FormatTime(nextNewDay), nextNewDay.Unix(),
			)
		})

		s.ChannelMessageSendEmbed(TFLOW_CHANNEL_ID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Ruin Events [%d]", count),
			Description: strings.Join(desc, "\n\n"),
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
		desc := parallel.Map(diff, func(t oapi.TownInfo, _ int) string {
			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			chunks := utils.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := utils.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

			return fmt.Sprintf(
				"`%s` was deleted. Located at %s.\nFounder: `%s` %sG %s Chunks",
				t.Name, locationLink, t.Founder, balance, chunks,
			)
		})

		s.ChannelMessageSendEmbed(TFLOW_CHANNEL_ID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Fall Events [%d]", count),
			Description: strings.Join(desc, "\n\n"),
			Color:       discordutil.RED,
		})
	}
}

func TrySendLeftJoinedNotif(s *discordgo.Session, towns, staleTowns []oapi.TownInfo, townless, residents oapi.EntityList) {
	left, joined := CalcLeftJoined(towns, staleTowns, townless, residents)

	leftCount := len(left)
	joinedCount := len(joined)
	if (leftCount + joinedCount) > 0 {
		s.ChannelMessageSendEmbed("1420108251437207682", &discordgo.MessageEmbed{
			Color: discordutil.DARK_GREEN,
			Title: "Player Flow | Town Join/Leave Events",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   fmt.Sprintf("%s Became townless [%d]", shared.EMOJIS.EXIT, leftCount),
					Value:  strings.Join(left, "\n\n"),
					Inline: true,
				},
				{
					Name:   fmt.Sprintf("%s Became a resident [%d]", shared.EMOJIS.ENTRY, joinedCount),
					Value:  strings.Join(joined, "\n\n"),
					Inline: true,
				},
			},
		})
	}
}

//#endregion

func CalcLeftJoined(towns, staleTowns []oapi.TownInfo, townless, residents oapi.EntityList) (left, joined []string) {
	// For resident -> town lookup (stale)
	staleResMap := make(map[string]oapi.TownInfo)
	for _, t := range staleTowns {
		for _, r := range t.Residents {
			staleResMap[r.UUID] = t
		}
	}

	// For resident -> town lookup (fresh not stale)
	resMap := make(map[string]oapi.TownInfo)
	for _, t := range towns {
		for _, r := range t.Residents {
			resMap[r.UUID] = t
		}
	}

	// who left
	for uuid, town := range staleResMap {
		if _, ok := resMap[uuid]; !ok {
			name, ok := townless[uuid]
			if !ok {
				continue // Left a town but not townless. Likely purged?
			}

			nation := "No Nation"
			if town.Nation.Name != nil {
				nation = *town.Nation.Name
			}

			ruined := lo.Ternary(town.Status.Ruined, ":white_check_mark:", ":x:")
			overclaimable := lo.Ternary(
				town.Status.Overclaimed && !town.Status.HasOverclaimShield,
				":white_check_mark:", ":x:",
			)

			left = append(joined, utils.HumanizedSprintf(
				"`%s` left %s (**%s**)\nMayor: `%s`, Balance: `%0.0f`G %s\nRuined %s Overclaimable %s",
				name, town.Name, nation,
				town.Mayor.Name, town.Bal(), shared.EMOJIS.GOLD_INGOT,
				ruined, overclaimable,
			))
		}
	}

	// who joined
	for uuid, town := range resMap {
		if _, ok := staleResMap[uuid]; !ok {
			name := residents[uuid]

			nation := "No Nation"
			if town.Nation.Name != nil {
				nation = *town.Nation.Name
			}

			ruined := lo.Ternary(town.Status.Ruined, ":white_check_mark:", ":x:")
			overclaimable := lo.Ternary(
				town.Status.Overclaimed && !town.Status.HasOverclaimShield,
				":white_check_mark:", ":x:",
			)

			joined = append(joined, utils.HumanizedSprintf(
				"`%s` joined %s (**%s**)\nMayor: `%s`, Balance: `%0.0f`G %s\nRuined %s Overclaimable %s",
				name, town.Name, nation,
				town.Mayor.Name, town.Bal(), shared.EMOJIS.GOLD_INGOT,
				ruined, overclaimable,
			))
		}
	}

	return
}
