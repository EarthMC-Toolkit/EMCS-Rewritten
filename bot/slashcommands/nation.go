package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
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
				discordutil.RequiredStringOption("name", "The name of the nation to query.", 2, 36),
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
