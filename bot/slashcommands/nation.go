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
	}
}

func (cmd NationCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		nationNameArg := opt.GetOption("name").StringValue()
		_, err := executeQueryNation(s, i.Interaction, nationNameArg)
		return err
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
	case "query":
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

	var matches []oapi.NationInfo
	if focused == "" {
		nations := nationStore.Values()

		// Sort alphabetically by Name.
		// TODO: Sort by nation rank first instead.
		sort.Slice(nations, func(i, j int) bool {
			return strings.ToLower(nations[i].Name) < strings.ToLower(nations[j].Name)
		})

		matches = nations
	} else {
		keyLower := strings.ToLower(focused)
		matches = nationStore.FindMany(func(n oapi.NationInfo) bool {
			if n.Name != "" && strings.Contains(strings.ToLower(n.Name), keyLower) {
				return true
			}
			if n.UUID != "" && n.UUID == focused {
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

	choices := discordutil.CreateAutocompleteChoices(matches, func(n oapi.NationInfo) (string, string) {
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
	var nation *oapi.NationInfo

	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		nation, err = getNationFromOAPI(nationName)
		if err != nil {
			return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("DB error occurred and the OAPI failed during fallback!?```%s```", err))
		}
	} else {
		if len(nationStore.Keys()) == 0 {
			return discordutil.FollowupContentEphemeral(s, i, "The nation database is currently empty. This is unusual, but may resolve itself.")
		}

		nation, _ = nationStore.FindFirst(func(info oapi.NationInfo) bool {
			return strings.EqualFold(nationName, info.Name) || nationName == info.UUID
		})
	}

	if nation == nil {
		return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Nation `%s` does not seem to exist.", nationName))
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

	embed := shared.NewNationEmbed(*nation)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func getNationFromOAPI(nationName string) (*oapi.NationInfo, error) {
	nations, err := oapi.QueryNations(strings.ToLower(nationName))
	if err != nil {
		return nil, err
	}

	if len(nations) == 0 {
		return nil, nil
	}

	n := nations[0]
	return &n, nil
}
