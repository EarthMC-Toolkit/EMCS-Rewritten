package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
		_, err := executeQueryTown(s, i.Interaction, townNameArg)
		return err
	}
	if opt := cdata.GetOption("list"); opt != nil {
		return executeListTowns(s, i.Interaction)
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

func executeQueryTown(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	var town *oapi.TownInfo

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		town, err = oapi.QueryTown(townName)
		if err != nil {
			return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("A database error occurred and the API failed during fallback!?```%s```", err))
		}
	} else {
		if len(townStore.Keys()) == 0 {
			return discordutil.FollowupContentEphemeral(s, i, "The town database is currently empty. This is unusual, but may resolve itself.")
		}

		town, _ = townStore.Find(func(info oapi.TownInfo) bool {
			return strings.EqualFold(townName, info.Name) || townName == info.UUID
		})
	}

	if town == nil {
		return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Town `%s` does not seem to exist.", townName))
	}

	embed := embeds.NewTownEmbed(*town)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func executeListTowns(s *discordgo.Session, i *discordgo.Interaction) error {
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
