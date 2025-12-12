package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"sort"
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
			Options:     AppCommandOpts{},
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
	if focused == "" {
		towns := townStore.Values()

		// Sort alphabetically by Name.
		// TODO: Sort by town rank first instead.
		sort.Slice(towns, func(i, j int) bool {
			return strings.ToLower(towns[i].Name) < strings.ToLower(towns[j].Name)
		})

		matches = towns
	} else {
		keyLower := strings.ToLower(focused)
		matches = townStore.FindMany(func(t oapi.TownInfo) bool {
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

	choices := discordutil.CreateAutocompleteChoices(matches, func(t oapi.TownInfo) (string, string) {
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
		town, err = getTownFromOAPI(townName)
		if err != nil {
			return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("A database error occurred and the API failed during fallback!?```%s```", err))
		}
	} else {
		if len(townStore.Keys()) == 0 {
			return discordutil.FollowupContentEphemeral(s, i, "The town database is currently empty. This is unusual, but may resolve itself.")
		}

		town, _ = townStore.FindFirst(func(info oapi.TownInfo) bool {
			return strings.EqualFold(townName, info.Name) || townName == info.UUID
		})
	}

	if town == nil {
		return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Town `%s` does not seem to exist.", townName))
	}

	embed := shared.NewTownEmbed(*town)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func getTownFromOAPI(townName string) (*oapi.TownInfo, error) {
	towns, err := oapi.QueryTowns(strings.ToLower(townName))
	if err != nil {
		return nil, err
	}

	if len(towns) == 0 {
		return nil, nil
	}

	t := towns[0]
	return &t, nil
}
