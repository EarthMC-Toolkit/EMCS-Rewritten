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
	"emcsrw/bot/scheduler"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils"
	"emcsrw/utils/config"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/logutil"

	"github.com/bwmarrin/discordgo"
	colour "github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

// max amount of messages to fetch from news channel during its scheduled task.
// should be enough to cover at least a few days/weeks of news depending on activity.
const NEWS_CHANNEL_MAX_FETCH = 500

// VoteParty notification tracking.
var vpThresholds = [...]int{500, 300, 150, 50}
var vpNotified = make(map[int]bool) // Key is each threshold, value is whether notified or not.
var vpLastRemaining int
var vpLastCheck time.Time

var readyOnce sync.Once // This prevents running tasks more than once if OnReady is called multiple times.

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as: %s\n", s.State.User.Username)

	readyOnce.Do(func() {
		mdb, err := database.Get(shared.ACTIVE_MAP)
		if err != nil {
			fmt.Printf("\n%s\n%v", colour.RedString("[OnReady]: wtf happened? error fetching db:"), err)
			return
		}

		scheduler.Instance.Schedule("DataUpdate", func() { dataUpdateTask(s, mdb) }, true, 30*time.Second)
		scheduler.Instance.Schedule("ServerInfo", func() { serverInfoTask(s, mdb) }, true, 1*time.Minute)

		if cid, err := config.GetEnviroVar("NEWS_CHANNEL_ID"); err == nil {
			scheduler.Instance.Schedule("NewsEntries", func() { newsTask(s, cid, mdb) }, true, 2*time.Minute)
		} else {
			logutil.Printf(logutil.YELLOW, "\nWARNING | NEWS_CHANNEL_ID not set. Skipped scheduling of news retrieval task.\n")
		}

		// TODO: Create a scheduled task that loops through alliances, removing nations that no longer exist.
	})
}

func dataUpdateTask(s *discordgo.Session, mdb *database.Database) {
	fmt.Println()
	logutil.Logln(logutil.WHITE, "[OnReady]: Running data update task...")

	start := time.Now()
	townList, staleTownList, townless, residents, err := UpdateData(mdb)

	fmt.Println() // use \n without log.Printf messing up date/time
	if err != nil {
		logutil.Logf(logutil.RED, "[OnReady]: Failed data update task.\n%s\n", err)
	} else {
		elapsed := time.Since(start)
		logutil.Logf(logutil.GREEN, "[OnReady]: Finished data update task. Took: %s\n", elapsed.String())
	}

	towns := lo.MapToSlice(townList, func(_ string, t oapi.TownInfo) oapi.TownInfo { return t })
	staleTowns := lo.MapToSlice(staleTownList, func(_ string, t oapi.TownInfo) oapi.TownInfo { return t })

	cid, err := config.GetEnviroVar("TFLOW_CHANNEL_ID")
	if err == nil {
		// TODO: ADD SOME SORT OF CHECK SO THEY CANT USE EMCS TO SPAM RANDOM CHANNELS!!!
		// Town flow event notifications sent to channel TFLOW_CHANNEL_ID.
		TrySendCreatedNotif(s, cid, towns, staleTowns)
		TrySendRenamedNotif(s, cid, townList, staleTowns)
		TrySendRuinedNotif(s, cid, townList, staleTowns)
		TrySendFallenNotif(s, cid, towns, staleTowns)
	} else {
		logutil.Printf(logutil.YELLOW, "\nWARNING | TFLOW_CHANNEL_ID not set. Skipping town flow event notifications.\n")
	}

	cid, err = config.GetEnviroVar("PFLOW_CHANNEL_ID")
	if err == nil {
		// Player flow event notifications sent to channel PFLOW_CHANNEL_ID.
		TrySendLeftJoinedNotif(s, cid, towns, staleTowns, townless, residents)
	} else {
		logutil.Printf(logutil.YELLOW, "\nWARNING | PFLOW_CHANNEL_ID not set. Skipping player flow event notifications.\n")
	}
}

func serverInfoTask(s *discordgo.Session, mdb *database.Database) {
	serverStore, err := database.GetStore(mdb, database.SERVER_STORE)
	if err != nil {
		logutil.Printf(logutil.RED, "\nERROR | cannot schedule serverinfo task:\n\t%s", err)
		return
	}
	if info, err := serverStore.SetKeyFunc("info", func() (oapi.ServerInfo, error) {
		info, err := oapi.QueryServer().Execute()
		return info, err
	}); err == nil {
		cid, err := config.GetEnviroVar("VP_CHANNEL_ID")
		if err != nil {
			logutil.Printf(logutil.YELLOW, "\nWARNING | VP_CHANNEL_ID not set. Skipping VoteParty notifications.\n")
		} else {
			TrySendVotePartyNotif(s, cid, info.VoteParty)
		}

		if err := serverStore.WriteSnapshot(); err != nil {
			logutil.Printf(logutil.RED, "\nERROR | server store failed to write snapshot:\n\t%s", err)
		}
	}
}

type NewsEntryMap = map[database.NewsMessageID]database.NewsEntry

func newsTask(s *discordgo.Session, channelID string, mdb *database.Database) {
	newsStore, err := database.GetStore(mdb, database.NEWS_STORE)
	if err != nil {
		logutil.Printf(logutil.RED, "\nERROR | cannot schedule news task:\n\t%s", err)
		return
	}

	newsMsgs, err := discordutil.FetchMessages(s, channelID, NEWS_CHANNEL_MAX_FETCH)
	if err != nil {
		logutil.Printf(logutil.RED, "\nERROR | news task failed to fetch messages:\n\t%s", err)
		return
	}

	entries := database.MessagesToNewsEntries(newsMsgs)
	for id, entry := range entries {
		newsStore.Set(id, entry)
	}

	if err := newsStore.WriteSnapshot(); err != nil {
		logutil.Printf(logutil.RED, "\nERROR | news store failed to write snapshot:\n\t%s", err)
	}
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
	logutil.Printf(logutil.HIDDEN, "DEBUG | Stale towns: %d\n", len(staleTowns))

	townList, err := townStore.OverwriteFunc(false, func() (map[string]oapi.TownInfo, error) {
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

	entityStore.SetKeyFunc("residentlist", func() (entities oapi.EntityList, err error) {
		return residentList, nil
	})

	nationList, _ := nationStore.OverwriteFunc(false, func() (map[string]oapi.NationInfo, error) {
		uuids := lo.Keys(nlist)
		res, _, _ := oapi.QueryNations(uuids...).ExecuteConcurrent()
		return lo.SliceToMap(res, func(n oapi.NationInfo) (string, oapi.NationInfo) {
			return n.UUID, n
		}), nil
	})
	//endregion

	//region ============ SPLIT RESIDENTS FROM TOWNLESS ============
	players, err := oapi.QueryList(oapi.ENDPOINT_PLAYERS).Execute()
	if err != nil {
		return townList, staleTowns, nil, residentList, err
	}

	townlessList, _ := entityStore.SetKeyFunc("townlesslist", func() (oapi.EntityList, error) {
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
			playerTownLookup[r.UUID] = &town
		}
	}

	playerNationLookup := make(map[string]*oapi.NationInfo, len(residentList))
	for _, nation := range nationList {
		for _, r := range nation.Residents {
			playerNationLookup[r.UUID] = &nation
		}
	}

	playerStore.OverwriteFunc(false, func() (map[string]database.BasicPlayer, error) {
		playersMap := make(map[string]database.BasicPlayer)
		for uuid, name := range townlessList {
			playersMap[uuid] = database.NewBasicPlayerEntity(uuid, name)
		}
		for uuid, name := range residentList {
			bp := database.NewBasicPlayerEntity(uuid, name)
			rank := database.RankTypeResident

			// Get player town by their UUID. While the town should always exist,
			// this prevents a potential panic and keeps them townless.
			if t, ok := playerTownLookup[uuid]; ok {
				bp.Town = &t.Entity
				if t.Mayor.UUID == uuid {
					rank = database.RankTypeMayor
				}

				if t.Nation.UUID != nil {
					bp.Nation = &oapi.Entity{Name: *t.Nation.Name, UUID: *t.Nation.UUID}
					if n, ok := playerNationLookup[uuid]; ok {
						if n.King.UUID == uuid {
							rank = database.RankTypeLeader
						}
					}
				}
			}

			bp.Rank = &rank
			playersMap[uuid] = bp
		}

		return playersMap, nil
	})

	logutil.Printf(logutil.HIDDEN, "\nDEBUG | Towns: %d, Nations: %d", len(townList), len(nationList))
	logutil.Printf(logutil.HIDDEN, "\nDEBUG | Total Players: %d, Residents: %d, Townless: %d", len(players), len(residentList), len(townlessList))

	return townList, staleTowns, townlessList, residentList, err
}

// #region Channel notifs
// TODO: Increase sample size from 2 (last check and current) with a sliding window for better rate/ETA accuracy.
//
// For example, recording the last 15 minutes would be done using 15 samples, each sample at 60 second intervals.
func TrySendVotePartyNotif(s *discordgo.Session, channelID string, vp oapi.ServerVoteParty) {
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
			if msg, err := s.ChannelMessageSend(channelID, content); err == nil {
				s.ChannelMessageCrosspost(channelID, msg.ID)
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

func TrySendRenamedNotif(s *discordgo.Session, channelID string, towns map[string]oapi.TownInfo, staleTowns []oapi.TownInfo) {
	desc := make([]string, 0)
	for _, old := range staleTowns {
		cur, ok := towns[old.UUID]
		if !ok || cur.Name == old.Name {
			continue // skip if deleted or same name
		}

		spawn := cur.Coordinates.Spawn
		locationLink := fmt.Sprintf(
			"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)",
			spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
		)

		chunks := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, cur.Size())
		balance := logutil.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, cur.Bal())
		desc = append(desc, fmt.Sprintf(
			"`%s` was renamed to `%s`.\nLocated at %s.\nFounder: `%s` %sG %s Chunks",
			old.Name, cur.Name, locationLink, cur.Founder, balance, chunks,
		))
	}

	if len(desc) > 0 {
		_, err := s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Rename Events [%d]", len(desc)),
			Description: strings.Join(desc, "\n\n"),
			Color:       discordutil.AQUA,
		})
		if err != nil {
			logutil.Logf(logutil.RED, "error sending town flow rename event:\n%v", err)
		}
	}
}

func TrySendCreatedNotif(s *discordgo.Session, channelID string, towns []oapi.TownInfo, staleTowns []oapi.TownInfo) {
	diff, _ := utils.DifferenceBy(towns, staleTowns, func(t oapi.TownInfo) string {
		return t.UUID
	})

	count := len(diff)
	if count > 0 {
		desc := parallel.Map(diff, func(t oapi.TownInfo, _ int) string {
			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			chunks := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := logutil.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

			openEmoji := lo.Ternary(t.Status.Open, shared.EMOJIS.CIRCLE_CHECK, shared.EMOJIS.CIRCLE_CROSS)
			outsidersEmoji := lo.Ternary(t.Status.CanOutsidersSpawn, shared.EMOJIS.CIRCLE_CHECK, shared.EMOJIS.CIRCLE_CROSS)

			return fmt.Sprintf(
				"`%s` was created. Located at %s.\nFounder: `%s` %sG %s Chunks %s Open %s Outsiders Can Spawn",
				t.Name, locationLink, t.Founder, balance, chunks, openEmoji, outsidersEmoji,
			)
		})

		logutil.Printf(logutil.HIDDEN, "\nDEBUG | Town Flow Channel ID: %s\n", channelID)
		_, err := s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Creation Events [%d]", count),
			Description: strings.Join(desc, "\n\n"),
			Color:       discordutil.GREEN,
		})
		if err != nil {
			logutil.Logf(logutil.RED, "error sending town flow creation event:\n%v", err)
		}
	}
}

func TrySendRuinedNotif(s *discordgo.Session, channelID string, towns map[string]oapi.TownInfo, staleTowns []oapi.TownInfo) {
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
			chunks := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := logutil.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

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

		_, err := s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Ruin Events [%d]", count),
			Description: strings.Join(desc, "\n\n"),
			Color:       discordutil.DARK_GOLD,
		})
		if err != nil {
			logutil.Logf(logutil.RED, "error sending town flow ruin event:\n%v", err)
		}
	}
}

func TrySendFallenNotif(s *discordgo.Session, channelID string, towns []oapi.TownInfo, staleTowns []oapi.TownInfo) {
	diff, _ := utils.DifferenceBy(staleTowns, towns, func(t oapi.TownInfo) string {
		return t.UUID
	})

	count := len(diff)
	if count > 0 {
		desc := parallel.Map(diff, func(t oapi.TownInfo, _ int) string {
			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			chunks := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := logutil.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())
			return fmt.Sprintf(
				"`%s` was deleted. Located at %s.\nFounder: `%s` %sG %s Chunks",
				t.Name, locationLink, t.Founder, balance, chunks,
			)
		})

		_, err := s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Town Flow | Fall Events [%d]", count),
			Description: strings.Join(desc, "\n\n"),
			Color:       discordutil.RED,
		})
		if err != nil {
			logutil.Logf(logutil.RED, "error sending town flow fall event:\n%v", err)
		}
	}
}

func TrySendLeftJoinedNotif(
	s *discordgo.Session, channelID string,
	towns, staleTowns []oapi.TownInfo,
	townless, residents oapi.EntityList,
) {
	left, joined := CalcLeftJoined(towns, staleTowns, townless, residents)
	if (len(left) + len(joined)) < 1 {
		return
	}

	s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
		Color: discordutil.DARK_GREEN,
		Title: "Player Flow | Town Join/Leave Events",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("%s Became townless [%d]", shared.EMOJIS.EXIT, len(left)),
				Value:  strings.Join(left, "\n\n"),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("%s Became a resident [%d]", shared.EMOJIS.ENTRY, len(joined)),
				Value:  strings.Join(joined, "\n\n"),
				Inline: true,
			},
		},
	})
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

			left = append(joined, logutil.HumanizedSprintf(
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

			joined = append(joined, logutil.HumanizedSprintf(
				"`%s` joined %s (**%s**)\nMayor: `%s`, Balance: `%0.0f`G %s\nRuined %s Overclaimable %s",
				name, town.Name, nation,
				town.Mayor.Name, town.Bal(), shared.EMOJIS.GOLD_INGOT,
				ruined, overclaimable,
			))
		}
	}

	return
}
