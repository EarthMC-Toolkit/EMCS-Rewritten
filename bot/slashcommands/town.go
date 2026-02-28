package slashcommands

import (
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const PURGE_DAYS = 42
const PURGE_DAYS_SEC = uint64(PURGE_DAYS * 24 * 3600) // PURGE_DAYS as seconds
const UINT64_MAX = ^uint64(0)

type TownCommand struct{}

func (cmd TownCommand) Name() string { return "town" }
func (cmd TownCommand) Description() string {
	return "Base command for town related subcommands."
}

func (cmd TownCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about a town. Similar to /t in-game.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the town to query.", 2, 36, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "activity",
			Description: "See the last online and purge date of each resident in a town.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the town to query.", 2, 40, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "online",
			Description: "Query the online status of a town's residents. Alias of /online town",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the town to query.", 2, 40, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "Sends a paginator enabling navigation through all existing towns.",
		},
	}
}

func (cmd TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		townNameArg := opt.GetOption("name").StringValue()
		_, err := executeTownQuery(s, i.Interaction, townNameArg)
		return err
	}
	if opt := cdata.GetOption("activity"); opt != nil {
		townNameArg := opt.GetOption("name").StringValue()
		_, err := executeTownActivity(s, i.Interaction, townNameArg)
		if err != nil {
			discordutil.ReplyWithError(s, i.Interaction, err)
			return err
		}
	}
	if opt := cdata.GetOption("online"); opt != nil {
		townNameArg := opt.GetOption("name").StringValue()
		return executeOnlineTown(s, i.Interaction, townNameArg)
	}
	if opt := cdata.GetOption("list"); opt != nil {
		return executeTownList(s, i.Interaction)
	}

	return nil
}

func (cmd TownCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error {
	cdata := i.ApplicationCommandData()
	if len(cdata.Options) == 0 {
		return nil
	}

	// top-level sub cmd or group
	subCmd := cdata.Options[0]
	switch subCmd.Name {
	case "activity":
		fallthrough
	case "online":
		fallthrough
	case "query":
		return townNameAutocomplete(s, i, cdata)
	}

	return nil
}

func townNameAutocomplete(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	focused, ok := discordutil.GetFocusedValue[string](cdata.Options)
	if !ok {
		return fmt.Errorf("town autocomplete error: focused value could not be cast as string")
	}

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	var matches []oapi.TownInfo
	if strings.TrimSpace(focused) == "" {
		towns := townStore.Values()
		matches = utils.MultiKeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
			{Compare: func(a, b oapi.TownInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			{Compare: func(a, b oapi.TownInfo) bool { return a.Size() > b.Size() }},
		})
	} else {
		keyLower := strings.ToLower(focused)
		matches = townStore.FindAll(func(t oapi.TownInfo) bool {
			if t.Name != "" && strings.Contains(strings.ToLower(t.Name), keyLower) {
				return true
			}
			if t.UUID != "" && t.UUID == focused {
				return true
			}

			return false
		})
	}

	// truncate to Discord limit
	if len(matches) > discordutil.AUTOCOMPLETE_CHOICE_LIMIT {
		limit := min(len(matches), discordutil.AUTOCOMPLETE_CHOICE_LIMIT)
		matches = matches[:limit]
	}

	choices := discordutil.CreateAutocompleteChoices(matches, func(t oapi.TownInfo, _ int) (string, string) {
		display := t.Name
		if t.Status.Capital {
			display = "⭐ " + t.Name
		}
		if t.Status.HasNation {
			display += fmt.Sprintf(" (%s)", *t.Nation.Name)
		}

		return display, t.Name
	})

	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

func executeTownQuery(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	town, err := tryGetTown(townName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	embed := embeds.NewTownEmbed(*town)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func executeTownList(s *discordgo.Session, i *discordgo.Interaction) error {
	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	townCount := townStore.Count()
	if townCount == 0 {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No towns seem to exist? Something may have gone wrong with the database or town store.",
		})

		return err
	}

	towns := townStore.Values()
	utils.MultiKeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
		{Compare: func(a, b oapi.TownInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
		{Compare: func(a, b oapi.TownInfo) bool { return a.Size() > b.Size() }},
	})

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i, townCount, perPage)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(townCount)

		townStrings := []string{}
		for idx, t := range towns[start:end] {
			nationName := "No Nation"
			if t.Status.HasNation {
				nationName = *t.Nation.Name
			}

			size := utils.HumanizedSprintf("`%d`/`%d` %s (Worth `%d` %s)",
				t.Size(), t.MaxSize(), shared.EMOJIS.CHUNK,
				t.Worth(), shared.EMOJIS.GOLD_INGOT,
			)

			balance := utils.HumanizedSprintf("`%0.f`G %s", t.Bal(), shared.EMOJIS.GOLD_INGOT)
			residents := utils.HumanizedSprintf("`%d`", len(t.Residents))

			overclaimed := shared.EMOJIS.CIRCLE_CROSS
			if t.Status.Overclaimed {
				overclaimed = shared.EMOJIS.CIRCLE_CHECK
			}

			overclaimShield := shared.EMOJIS.SHIELD_RED
			if t.Status.HasOverclaimShield {
				overclaimShield = shared.EMOJIS.SHIELD_GREEN
			}

			overclaim := fmt.Sprintf("%s / %s", overclaimed, overclaimShield)
			townStrings = append(townStrings, fmt.Sprintf(
				"%d. %s (**%s**)\nMayor: `%s`\nResidents: %s\nBalance: %s\nSize: %s\nOverclaimed/Has Shield: %s",
				start+idx+1, t.Name, nationName, t.Mayor.Name, residents, balance, size, overclaim,
			))
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("[%d] List of Towns | %s", townCount, pageStr),
			Description: strings.Join(townStrings, "\n\n"),
			Color:       discordutil.GREEN,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
}

func executeTownActivity(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	town, err := tryGetTown(townName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	// TODO: Maybe do this inside of PageFunc using residents on current page
	residents, errs, _ := oapi.QueryConcurrentEntities(oapi.QueryPlayers, town.Residents)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	count := len(residents)
	slices.SortFunc(residents, func(a, b oapi.PlayerInfo) int {
		at, bt := a.Timestamps.LastOnline, b.Timestamps.LastOnline
		av, bv := UINT64_MAX, UINT64_MAX
		if at != nil {
			av = *at
		}
		if bt != nil {
			bv = *bt
		}

		return cmp.Compare(av, bv)
	})

	perPage := 10
	paginator := discordutil.NewInteractionPaginator(s, i, count, perPage).
		WithTimeout(5 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		now := uint64(time.Now().Unix())
		content := strings.Builder{}
		for _, res := range residents[start:end] {
			lo := res.Timestamps.LastOnline
			if lo == nil {
				fmt.Fprintf(&content, "**%s** - Unknown\n", res.Name)
				continue
			}

			//daysOffline := (now - *lo/1000) / 86400
			purgeTimeStr := formattedPurgeTime(now, *lo/1000)
			balanceStr := utils.HumanizedSprintf("%s `%d`G", shared.EMOJIS.GOLD_INGOT, int(res.Stats.Balance))

			fmt.Fprintf(&content,
				"**%s** (%s) - Online <t:%d:R>. Purges %s. %s\n",
				res.Name, res.GetRankOrRole(), *lo/1000, purgeTimeStr, balanceStr,
			)
		}

		data.Content = content.String()
		if paginator.TotalPages() > 1 {
			data.Content += fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages())
		}
	}

	// chinese motorcycle
	return nil, paginator.Start()
}

// Given the current time and last online, this func outputs a formatted time
// according to the remaining time until a player is offline for PURGE_DAYS.
func formattedPurgeTime(now, lastOnline uint64) string {
	purgeAt := lastOnline + PURGE_DAYS_SEC
	if now >= purgeAt {
		return "`at pending newday`"
	}

	return fmt.Sprintf("<t:%d:R>", purgeAt)
}

// Attempts to get a town from the town store for the current map, unless that
// fails in which case we fall back to querying it via the OAPI.
//
// If all fails, the town will be nil and an appropriate error message will be returned.
func tryGetTown(townName string) (*oapi.TownInfo, error) {
	var town *oapi.TownInfo

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		if town, err = oapi.QueryTown(townName); err != nil {
			return nil, fmt.Errorf("A database error occurred and the API failed during fallback!?```%s```", err)
		}
	} else {
		if townStore.Count() == 0 {
			return nil, fmt.Errorf("The town database is currently empty. This is unusual, but may resolve itself.")
		}

		town, _ = townStore.Find(func(info oapi.TownInfo) bool {
			return strings.EqualFold(townName, info.Name) || townName == info.UUID
		})
	}

	if town == nil {
		return nil, fmt.Errorf("Town `%s` does not seem to exist.", townName)
	}

	return town, nil
}
