package slashcommands

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/logutil"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
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
			Options: AppCommandOpts{
				discordutil.StringOption("sort", "Optional town list sorting. Without this, towns are sorted by residents -> size.", nil, nil,
					//discordutil.Choice("None", "none"),                 // "No list sorting. The entropy enjoyer's choice."
					discordutil.Choice("Alphabetical", "alphabetical"), // "Sort the list alphabetically by name."
					discordutil.Choice("Founded", "founded"),           // "Sort the list by date founded. Oldest -> Newest."
					discordutil.Choice("Residents", "residents"),       // "Sort the list solely by the number of residents."
					discordutil.Choice("Size", "size"),                 // "Sort the list solely by size (chunks claimed)."
					discordutil.Choice("Balance", "balance"),           // "Sort the list solely by balance."
					discordutil.Choice("Ruined", "ruined"),             // "Sort the list by for sale status. For Sale (highest-lowest) -> Not For Sale."
					discordutil.Choice("Overclaimed", "overclaimed"),   // "Sort the list by overclaim status. Oldest -> Newest."
					discordutil.Choice("For Sale", "for-sale"),         // "Sort the list by for sale status. For Sale (highest-lowest) -> Not For Sale."
					discordutil.Choice("Has Nation", "has-nation"),
					discordutil.Choice("Can Outsiders Spawn", "can-outsiders-spawn"), // "Sort the list by outsider spawn status. Enabled -> Not enabled."
					discordutil.Choice("Open", "open"),                               // "Sort the list by open status. Open -> Not open."
					discordutil.Choice("Public", "public"),                           // "Sort the list by public status. Public -> Not public."
					discordutil.Choice("Neutral", "neutral"),                         // "Sort the list by neutral status. Neutral -> Not neutral."
				),
				discordutil.AutocompleteStringOption("nation", "Filter by towns within a specified nation.", 2, 40, false),
				discordutil.StringOption("status", "Filter by towns with the specified status active.", nil, nil,
					discordutil.Choice("Ruined", "ruined"),
					discordutil.Choice("Overclaimed", "overclaimed"),
					discordutil.Choice("For Sale", "for-sale"),
					discordutil.Choice("Can Outsiders Spawn", "can-outsiders-spawn"),
					discordutil.Choice("Open", "open"),
					discordutil.Choice("Public", "public"),
					discordutil.Choice("Neutral", "neutral"),
				),
				discordutil.StringOption("flags", "Filter by towns with the specified flag toggled on.", nil, nil,
					discordutil.Choice("Explosions", "explosions"),
					discordutil.Choice("Mobs", "mobs"),
					discordutil.Choice("Fire", "fire"),
					discordutil.Choice("PVP", "pvp"),
				),
			},
		},
	}
}

func (cmd TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
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
	case "list":
		for _, opt := range subCmd.Options {
			if opt.Name == "nation" {
				return nationNameAutocomplete(s, i, cdata)
			}
		}
	case "query", "activity", "online":
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

	matches := []oapi.TownInfo{}
	focusedTrimmed := strings.TrimSpace(focused)

	if focusedTrimmed == "" {
		towns := townStore.Values()
		matches = utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
			{Compare: func(a, b oapi.TownInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			{Compare: func(a, b oapi.TownInfo) bool { return a.Size() > b.Size() }},
		})
	} else {
		focusedLower := strings.ToLower(focusedTrimmed)
		matches = townStore.FindAll(func(t oapi.TownInfo) bool {
			if t.Name != "" && strings.Contains(strings.ToLower(t.Name), focusedLower) {
				return true
			}
			if t.UUID != "" && t.UUID == focusedTrimmed {
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
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return nil, err
	}

	town, err := tryGetTown(mdb, townName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	msg := discordutil.NewMessageBuilder()
	msg.AddEmbed(embeds.NewTownEmbed(*town))
	if town.Discord != nil {
		discordEmoji := &discordgo.ComponentEmoji{Name: "discordlogo", ID: "1513955352608243923"}
		msg.AddButton("Join discord", discordgo.LinkButton, town.Discord, discordEmoji, nil)
	}
	if town.Wiki != "" {
		linkEmoji := &discordgo.ComponentEmoji{Name: "📰"}
		msg.AddButton("View wiki page", discordgo.LinkButton, &town.Wiki, linkEmoji, nil)
	}

	return discordutil.Followup(s, i, msg.WebhookData())
}

func executeTownList(s *discordgo.Session, i *discordgo.Interaction) error {
	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	if townStore.Count() == 0 {
		_, err := discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
			Content: "No towns seem to exist? Something may have gone wrong with the database or town store.",
		})

		return err
	}

	towns := townStore.Values()

	listOpt := i.ApplicationCommandData().GetOption("list")
	if opt := listOpt.GetOption("nation"); opt != nil {
		nationInput := opt.StringValue()

		// parse input so "Japan" and "Japan (Capital: Chiyoda)" both work by extracting the nation name before any parentheses,
		// since that's what the autocomplete displays and it seems reasonable to expect users to copy-paste from autocomplete results.
		// "((Japan))", "(Japan (blahblah))" etc. also need to work incase of stupid nation names.
		nationName := strings.TrimSpace(nationInput)
		for strings.HasPrefix(nationName, "(") && strings.HasSuffix(nationName, ")") {
			nationName = strings.TrimSpace(nationName[1 : len(nationName)-1])
		}
		if i := strings.IndexRune(nationName, '('); i >= 0 {
			nationName = strings.TrimSpace(nationName[:i])
		}

		towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
			if t.Nation.UUID == nil {
				return false
			}

			return strings.EqualFold(*t.Nation.Name, nationName)
		})
	}
	if opt := listOpt.GetOption("status"); opt != nil {
		switch opt.StringValue() {
		case "ruined":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.Ruined
			})
		case "overclaimed":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.Overclaimed
			})
		case "for-sale":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.ForSale
			})
		case "can-outsiders-spawn":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.CanOutsidersSpawn
			})
		case "open":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.Open
			})
		case "public":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.Public
			})
		case "neutral":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Status.Neutral
			})
		}
	}
	if opt := listOpt.GetOption("flags"); opt != nil {
		switch opt.StringValue() {
		case "explosions":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Perms.Flags.Explosions
			})
		case "mobs":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Perms.Flags.Mobs
			})
		case "fire":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Perms.Flags.Fire
			})
		case "pvp":
			towns = lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
				return t.Perms.Flags.PVP
			})
		}
	}
	if opt := listOpt.GetOption("sort"); opt != nil {
		switch opt.StringValue() {
		case "alphabetical":
			utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
				{Compare: func(a, b oapi.TownInfo) bool { return a.Name < b.Name }}, // ascending (A-Z)
			})
		case "residents":
			utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
				{Compare: func(a, b oapi.TownInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			})
		case "size":
			utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
				{Compare: func(a, b oapi.TownInfo) bool { return a.Size() > b.Size() }}, // descending (highest first)
			})
		case "founded":
			utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
				{Compare: func(a, b oapi.TownInfo) bool { return a.Timestamps.Registered < b.Timestamps.Registered }}, // ascending (oldest-newest)
			})
		case "balance":
			utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
				{Compare: func(a, b oapi.TownInfo) bool { return a.Bal() > b.Bal() }}, // descending (highest first)
			})
		case "ruined":
			utils.RankSortDescending(towns, func(t oapi.TownInfo) int {
				if !t.Status.Ruined {
					return 0
				}

				return int(*t.Timestamps.RuinedAt)
			})
		case "overclaimed":
			utils.RankSortAscending(towns, func(t oapi.TownInfo) int {
				switch {
				case t.Status.Overclaimed:
					return 0
				case t.Status.Overclaimed:
					return 1
				default:
					return 2
				}
			})
		case "for-sale":
			utils.RankSortDescending(towns, func(t oapi.TownInfo) int {
				if !t.Status.ForSale || t.Stats.ForSalePrice == nil {
					return 0
				}

				return int(*t.Stats.ForSalePrice * 100.0)
			})
		case "has-nation":
			utils.SortToggledOn(towns, func(t oapi.TownInfo) bool {
				return t.Status.HasNation
			})
		case "can-outsiders-spawn":
			utils.SortToggledOn(towns, func(t oapi.TownInfo) bool {
				return t.Status.CanOutsidersSpawn
			})
		case "open":
			utils.SortToggledOn(towns, func(t oapi.TownInfo) bool {
				return t.Status.Open
			})
		case "public":
			utils.SortToggledOn(towns, func(t oapi.TownInfo) bool {
				return t.Status.Public
			})
		case "neutral":
			utils.SortToggledOn(towns, func(t oapi.TownInfo) bool {
				return t.Status.Neutral
			})
		}
	} else {
		// No sort option provided, use default sort (residents -> size).
		utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{
			{Compare: func(a, b oapi.TownInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			{Compare: func(a, b oapi.TownInfo) bool { return a.Size() > b.Size() }},
		})
	}

	townCount := len(towns)

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

			size := logutil.HumanizedSprintf("`%d`/`%d` %s (Worth `%d`G %s)",
				t.Size(), t.MaxSize(), shared.EMOJIS.CHUNK,
				t.Worth(), shared.EMOJIS.GOLD_INGOT,
			)

			balance := logutil.HumanizedSprintf("`%0.f`G %s", t.Bal(), shared.EMOJIS.GOLD_INGOT)
			residents := logutil.HumanizedSprintf("`%d`", len(t.Residents))

			overclaimed := lo.Ternary(t.Status.Overclaimed, shared.EMOJIS.CIRCLE_CHECK, shared.EMOJIS.CIRCLE_CROSS)
			ruined := lo.Ternary(t.Status.Ruined, shared.EMOJIS.CIRCLE_CHECK, shared.EMOJIS.CIRCLE_CROSS)
			forsale := lo.Ternary(t.Status.ForSale, shared.EMOJIS.CIRCLE_CHECK, shared.EMOJIS.CIRCLE_CROSS)

			status := fmt.Sprintf("%s / %s / %s", overclaimed, ruined, forsale)

			// convert ms to sec for Discord timestamp
			foundedStr := fmt.Sprintf("Founded <t:%d:R>", t.Timestamps.Registered/1000)
			townStrings = append(townStrings, fmt.Sprintf(
				"%d. %s (**%s**) • %s\nMayor: `%s`\nResidents: %s\nSize: %s\nBalance: %s\nOverclaimed/Ruined/For Sale: %s",
				start+idx+1, t.Name, nationName, foundedStr, t.Mayor.Name, residents, size, balance, status,
			))
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		title := fmt.Sprintf("[%d] List of Towns | %s", townCount, pageStr)
		desc := strings.Join(townStrings, "\n\n")

		embed := discordutil.NewEmbedBuilder(&discordutil.GREEN, &title, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}

func executeTownActivity(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return nil, err
	}

	town, err := tryGetTown(mdb, townName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	ids := parallel.Map(town.Residents, func(e oapi.Entity, _ int) string {
		return e.UUID
	})

	// TODO: Maybe do this inside of PageFunc using residents on current page
	residents, errs, _ := oapi.QueryPlayers(ids...).ExecuteConcurrent()
	if len(errs) > 0 {
		errStr := fmt.Sprintf("Failed to query player data for residents of `%s`.```%s```", town.Name, errs[0].Error())
		return discordutil.FollowupContentEphemeral(s, i, errStr)
	}

	count := len(residents)
	slices.SortFunc(residents, func(a, b oapi.PlayerInfo) int {
		return utils.ComparePtr(b.Timestamps.LastOnline, a.Timestamps.LastOnline, UINT64_MAX) // sort by most active
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
			balanceStr := logutil.HumanizedSprintf("%s `%d`G", shared.EMOJIS.GOLD_INGOT, int(res.Stats.Balance))

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
func tryGetTown(mdb *database.Database, townName string) (*oapi.TownInfo, error) {
	var town *oapi.TownInfo

	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		town, err = api.QueryTown(townName)
		if err != nil {
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
