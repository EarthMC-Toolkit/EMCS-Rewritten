package slashcommands

import (
	"cmp"
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
	"github.com/samber/lo/parallel"
)

type NationCommand struct{}

func (cmd NationCommand) Name() string { return "nation" }
func (cmd NationCommand) Description() string {
	return "Base command for nation related subcommands."
}

func (cmd NationCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about a nation. Similar to /n in-game.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the nation to query.", 2, 36, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "activity",
			Description: "See the last online and purge date of each resident in a town.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the nation to query.", 2, 40, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "online",
			Description: "Query the online status of a nation's residents. Alias of /online nation",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the nation to query.", 2, 40, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "Sends a paginator enabling navigation through all existing nations.",
			Options: AppCommandOpts{
				discordutil.StringOption("sort", "Optional nation list sorting. Without this, nations are sorted by residents -> towns -> size.", nil, nil,
					//discordutil.Choice("None", "none"),                 //"No list sorting. The entropy enjoyer's choice."),
					discordutil.Choice("Alphabetical", "alphabetical"), //"Sort the list alphabetically by name."),
					discordutil.Choice("Residents", "residents"),       //"Sort the list solely by the number of residents."),
					discordutil.Choice("Towns", "towns"),               //"Sort the list solely by the number of towns."),
					discordutil.Choice("Size", "size"),                 //"Sort the list solely by size (chunks claimed)."),
					discordutil.Choice("Founded", "founded"),           //"Sort the list by date founded. Oldest -> Newest."),
					//discordutil.Choice("Towns Overclaimed", "towns-overclaimed"),   //"Sort the list by date founded. Oldest -> Newest."),
				),
			},
		},
	}
}

func (cmd NationCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		nationNameArg := opt.GetOption("name").StringValue()
		_, err := executeQueryNation(s, i.Interaction, nationNameArg)
		return err
	}
	if opt := cdata.GetOption("activity"); opt != nil {
		nationNameArg := opt.GetOption("name").StringValue()
		_, err := executeNationActivity(s, i.Interaction, nationNameArg)
		return err
	}
	if opt := cdata.GetOption("online"); opt != nil {
		nationNameArg := opt.GetOption("name").StringValue()
		return executeOnlineNation(s, i.Interaction, nationNameArg)
	}
	if opt := cdata.GetOption("list"); opt != nil {
		return executeListNations(s, i.Interaction)
	}

	return nil
}

func (cmd NationCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error {
	cdata := i.ApplicationCommandData()
	if len(cdata.Options) == 0 {
		return nil
	}

	// top-level sub cmd or group
	subCmd := cdata.Options[0]
	switch subCmd.Name {
	case "query", "activity", "online":
		return nationNameAutocomplete(s, i, cdata)
	}

	return nil
}

func nationNameAutocomplete(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	focused, ok := discordutil.GetFocusedValue[string](cdata.Options)
	if !ok {
		return fmt.Errorf("nation autocomplete error: focused value could not be cast as string")
	}

	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	matches := []oapi.NationInfo{}
	focusedTrimmed := strings.TrimSpace(focused)

	if focusedTrimmed == "" {
		// Sort by largest first.
		// TODO: This is pretty primitive and we should prefer database.GetRankedNations()
		// 		 when rank caching has been implemented.
		nations := nationStore.Values()
		matches = utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
			{Compare: func(a, b oapi.NationInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			{Compare: func(a, b oapi.NationInfo) bool { return a.NumTowns() > b.NumTowns() }},
			{Compare: func(a, b oapi.NationInfo) bool { return a.Size() > b.Size() }},
		})
	} else {
		focusedLower := strings.ToLower(focusedTrimmed)
		matches = nationStore.FindAll(func(n oapi.NationInfo) bool {
			if n.Name != "" && strings.Contains(strings.ToLower(n.Name), focusedLower) {
				return true
			}
			if n.UUID != "" && n.UUID == focusedTrimmed {
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

	choices := discordutil.CreateAutocompleteChoices(matches, func(n oapi.NationInfo, _ int) (string, string) {
		display := fmt.Sprintf("%s (Capital: %s)", n.Name, n.Capital.Name)
		return display, n.Name
	})

	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// func (cmd NationCommand) HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	if strings.HasPrefix(customID, "nation_relations") {
// 		return nil
// 	}

// 	return nil
// }

func executeQueryNation(s *discordgo.Session, i *discordgo.Interaction, nationName string) (*discordgo.Message, error) {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return nil, err
	}

	nation, err := tryGetNation(mdb, nationName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	// button := discordgo.Button{
	// 	CustomID: "nation_relations@" + nations[0].UUID,
	// 	Label:    "Show Relations",
	// 	Style:    discordgo.PrimaryButton,
	// }

	// row := discordgo.ActionsRow{
	// 	Components: []discordgo.MessageComponent{button},
	// }

	// return discordutil.Followup(s, i, &discordgo.WebhookParams{
	// 	Embeds:     []*discordgo.MessageEmbed{embed},
	// 	Components: []discordgo.MessageComponent{row},
	// })

	allianceStore, _ := database.GetStore(mdb, database.ALLIANCES_STORE)
	newsStore, _ := database.GetStore(mdb, database.NEWS_STORE)

	embed := embeds.NewNationEmbed(*nation, newsStore, allianceStore)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func executeListNations(s *discordgo.Session, i *discordgo.Interaction) error {
	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	if nationStore.Count() == 0 {
		_, err := discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
			Content: "No nations seem to exist? Something may have gone wrong with the database or nation store.",
		})

		return err
	}

	nations := nationStore.Values()

	listOpt := i.ApplicationCommandData().GetOption("list")
	if opt := listOpt.GetOption("sort"); opt != nil {
		switch opt.StringValue() {
		case "alphabetical":
			utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
				{Compare: func(a, b oapi.NationInfo) bool { return b.Name > a.Name }}, // ascending (A-Z)
			})
		case "residents":
			utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
				{Compare: func(a, b oapi.NationInfo) bool { return a.NumResidents() > b.NumResidents() }}, // descending
			})
		case "towns":
			utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
				{Compare: func(a, b oapi.NationInfo) bool { return a.NumTowns() > b.NumTowns() }}, // descending
			})
		case "size":
			utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
				{Compare: func(a, b oapi.NationInfo) bool { return a.Size() > b.Size() }}, // descending
			})
		case "founded":
			utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
				{Compare: func(a, b oapi.NationInfo) bool { return b.Timestamps.Registered > a.Timestamps.Registered }}, // ascending (oldest-newest)
			})
			// case "towns-overclaimed":
		}
	} else {
		// No sort option provided, use default sort (residents -> towns -> size).
		utils.KeySort(nations, []utils.KeySortOption[oapi.NationInfo]{
			{Compare: func(a, b oapi.NationInfo) bool { return a.NumResidents() > b.NumResidents() }},
			{Compare: func(a, b oapi.NationInfo) bool { return a.NumTowns() > b.NumTowns() }},
			{Compare: func(a, b oapi.NationInfo) bool { return a.Size() > b.Size() }},
		})
	}

	nationCount := len(nations)

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i, nationCount, perPage)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(nationCount)

		nationStrings := []string{}
		for idx, n := range nations[start:end] {
			foundedTs := n.Timestamps.Registered / 1000 // convert ms to sec for Discord timestamp
			balance := logutil.HumanizedSprintf("`%0.f`G %s", n.Bal(), shared.EMOJIS.GOLD_INGOT)
			towns := logutil.HumanizedSprintf("`%d`", n.Stats.NumTowns)
			residents := logutil.HumanizedSprintf("`%d`", n.Stats.NumResidents)
			size := logutil.HumanizedSprintf("`%d` %s (Worth `%d` %s)",
				n.Size(), shared.EMOJIS.CHUNK,
				n.Worth(), shared.EMOJIS.GOLD_INGOT,
			)

			nationStrings = append(nationStrings, fmt.Sprintf(
				"%d. %s (Capital: %s) • Founded <t:%d:R>\nLeader: `%s`\nTowns: %s\nResidents: %s\nBalance: %s\nSize: %s",
				start+idx+1, n.Name, n.Capital.Name, foundedTs, n.King.Name, towns, residents, balance, size,
			))
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("[%d] List of Nations | %s", nationCount, pageStr),
			Description: strings.Join(nationStrings, "\n\n"),
			Color:       discordutil.AQUA,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
}

func executeNationActivity(s *discordgo.Session, i *discordgo.Interaction, nationName string) (*discordgo.Message, error) {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return nil, err
	}

	nation, err := tryGetNation(mdb, nationName)
	if err != nil {
		return discordutil.FollowupContentEphemeral(s, i, err.Error())
	}

	ids := parallel.Map(nation.Residents, func(e oapi.Entity, _ int) string {
		return e.UUID
	})

	residents, errs, _ := oapi.QueryPlayers(ids...).ExecuteConcurrent()
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

	return nil, paginator.Start()
}

func tryGetNation(mdb *database.Database, nationName string) (*oapi.NationInfo, error) {
	var nation *oapi.NationInfo

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		nation, err = api.QueryNation(nationName)
		if err != nil {
			return nil, fmt.Errorf("DB error occurred and the OAPI failed during fallback!?```%s```", err)
		}
	} else {
		if len(nationStore.Keys()) == 0 {
			return nil, fmt.Errorf("The nation database is currently empty. This is unusual, but may resolve itself.")
		}

		nation, _ = nationStore.Find(func(info oapi.NationInfo) bool {
			return strings.EqualFold(nationName, info.Name) || nationName == info.UUID
		})
	}

	if nation == nil {
		return nil, fmt.Errorf("Nation `%s` does not seem to exist.", nationName)
	}

	return nation, nil
}
