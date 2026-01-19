package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var PLAYER_STATS_GROUP_ORDER = [...]string{
	"player_kills",
	"mob_kills",
	"deaths",

	"damage_taken",
	"damage_dealt",
	"damage_resisted",
	"damage_absorbed",
	"damage_dealt_resisted",
	"damage_dealt_absorbed",
	"damage_blocked_by_shield",

	"interact_with_crafting_table",
	"interact_with_smithing_table",
	"interact_with_cartography_table",
	"interact_with_furnace",
	"interact_with_blast_furnace",
	"interact_with_smoker",
	"interact_with_stonecutter",
	"interact_with_grindstone",
	"interact_with_anvil",
	"interact_with_loom",
	"interact_with_lectern",
	"interact_with_beacon",
	"interact_with_campfire",
	"interact_with_brewingstand",

	"inspect_dispenser",
	"inspect_dropper",
	"inspect_hopper",

	"open_enderchest",
	"open_chest",
	"open_barrel",
	"open_shulker_box",

	"traded_with_villager",
	"talked_to_villager",

	"total_world_time",
	"play_time",
	"sneak_time",
	"time_since_rest",
	"time_since_death",

	"raid_win",
	"raid_trigger",

	"walk_one_cm",
	"sprint_one_cm",
	"boat_one_cm",
	"swim_one_cm",
	"fall_one_cm",
	"walk_on_water_one_cm",
	"walk_under_water_one_cm",
	"horse_one_cm",
	"minecart_one_cm",
	"aviate_one_cm",
	"crouch_one_cm",
	"climb_one_cm",
	"pig_one_cm",
	"fly_one_cm",
	"strider_one_cm",

	"clean_banner",
	"clean_shulker_box",
	"clean_armor",

	"animals_bred",
	"fish_caught",

	"tune_noteblock",
	"play_noteblock",
	"play_record",
	"jump",
	"bell_ring",
	"eat_cake_slice",
	"sleep_in_bed",
	"trigger_trapped_chest",
	"fill_cauldron",
	"use_cauldron",
	"target_hit",
	"drop_count",
	"pot_flower",
	"enchant_item",
	"leave_game",
}

type ServerCommand struct{}

func (cmd ServerCommand) Name() string { return "server" }
func (cmd ServerCommand) Description() string {
	return "Base command for server related subcommands."
}

func (cmd ServerCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "info",
			Description: "Replies with information about the server (time, new day, vp).",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "player-stats",
			Description: "Sends a paginated list with the server's all-time player statistics.",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "sort",
					Description: "Sort order of the keys used before displaying the list.",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Grouped", Value: "grouped"},
						{Name: "Alphabetical", Value: "alphabetical"},
					},
				},
			},
		},
	}
}

func (cmd ServerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("info"); opt != nil {
		_, err := executeServerInfo(s, i.Interaction)
		return err
	}
	if opt := cdata.GetOption("player-stats"); opt != nil {
		sortArg := opt.GetOption("sort").StringValue()
		_, err := executeServerPlayerStats(s, i.Interaction, sortArg)
		return err
	}

	return nil
}

func executeServerInfo(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	serverStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.SERVER_STORE)
	if err != nil {
		return nil, err
	}

	info, err := serverStore.Get("info")
	if err != nil {
		log.Printf("failed to get server info from db:\n%v", err)
		return discordutil.FollowupContentEphemeral(s, i, "An error occurred retrieving server info from the database. Check the console.")
	}

	timePassed := info.Stats.Time
	serverTime := info.Timestamps.ServerTimeOfDay
	newDayTime := info.Timestamps.NewDayTime

	timestampsField := &discordgo.MessageEmbedField{
		Name: "Timestamps",
		Value: strings.Join([]string{
			fmt.Sprintf("Server Time Of Day: `%d`", serverTime),
			fmt.Sprintf("Time Passed In Current Day: `%d`", timePassed),
			fmt.Sprintf("New Day At: `%d`", newDayTime),
		}, "\n"),
		Inline: true,
	}

	vpTarget := info.VoteParty.Target
	vpRemaining := info.VoteParty.NumRemaining
	vpField := &discordgo.MessageEmbedField{
		Name:   "Vote Party",
		Value:  utils.HumanizedSprintf("Votes Completed/Target: `%d`/`%d`\nVotes Remaining: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
		Inline: true,
	}

	statsField := &discordgo.MessageEmbedField{
		Name: "Statistics",
		Value: strings.Join([]string{
			utils.HumanizedSprintf("Online: `%d`", info.Stats.NumOnlinePlayers),
			utils.HumanizedSprintf("Townless (Online/Total): `%d`/`%d`", info.Stats.NumOnlineNomads, info.Stats.NumNomads),
			utils.HumanizedSprintf("Residents: `%d`", info.Stats.NumResidents),
			utils.HumanizedSprintf("Towns: `%d`", info.Stats.NumTowns),
			utils.HumanizedSprintf("Nations: `%d`", info.Stats.NumNations),
			utils.HumanizedSprintf("Quarters: `%d`", info.Stats.NumQuarters),
		}, "\n"),
		Inline: false,
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Server Info",
		Fields: []*discordgo.MessageEmbedField{timestampsField, vpField, statsField},
		Color:  discordutil.BLURPLE,
		Footer: embeds.DEFAULT_FOOTER,
	}

	return discordutil.FollowupEmbeds(s, i, embed)
}

func executeServerPlayerStats(s *discordgo.Session, i *discordgo.Interaction, sortArg string) (*discordgo.Message, error) {
	pstats, err := oapi.QueryServerPlayerStats()
	if err != nil {
		log.Printf("failed to get server player stats:\n%v", err)
		return discordutil.FollowupContentEphemeral(s, i, "An error occurred retrieving server player stats. Check the console.")
	}

	count := len(pstats)
	keys := make([]string, 0, count)
	for k := range pstats {
		keys = append(keys, k)
	}

	if sortArg == "alphabetical" {
		sort.Strings(keys)
	}
	if sortArg == "grouped" {
		orderIndex := make(map[string]int, len(PLAYER_STATS_GROUP_ORDER))
		for i, k := range PLAYER_STATS_GROUP_ORDER {
			orderIndex[k] = i
		}

		sort.Slice(keys, func(i, j int) bool {
			ai, aOk := orderIndex[keys[i]]
			bj, bOk := orderIndex[keys[j]]
			if aOk && bOk {
				return ai < bj // ai blowjob
			}
			if aOk {
				return true
			}
			if bOk {
				return false
			}

			return false
		})
	}

	link := "See why [here](https://github.com/EarthMC/EMCAPI/blob/main/docs/global-player-stats.md#player-statistics-endpoint."
	outdated := "⚠️ As of `July 28, 2025`, the Official API no longer updates this data.\n" + link

	perPage := 20
	paginator := discordutil.NewInteractionPaginator(s, i, count, perPage).
		WithTimeout(5 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		desc := ""
		items := keys[start:end] // cur page items
		for _, k := range items {
			desc += fmt.Sprintf("%s: %d\n", k, pstats[k])
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("All-Time Player Statistics | %s", pageStr),
			Description: fmt.Sprintf("%s```%s```", outdated, desc),
			Footer:      embeds.DEFAULT_FOOTER,
			Color:       discordutil.DARK_GOLD,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return nil, paginator.Start()
}

// func executeServerTownyStats(s *discordgo.Session, i *discordgo.Interaction, sortArg string) (*discordgo.Message, error) {
// 	return nil, nil
// }
