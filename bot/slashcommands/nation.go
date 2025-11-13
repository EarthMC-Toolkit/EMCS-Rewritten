package slashcommands

import (
	"emcsrw/api/oapi"
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

func executeQueryNation(s *discordgo.Session, i *discordgo.Interaction, nationName string) (*discordgo.Message, error) {
	nations, err := oapi.QueryNations(strings.ToLower(nationName))
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving nation information :(")
	}

	if len(nations) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No nations retrieved. Nation `%s` does not seem to exist.", nationName))
	}

	embed := shared.NewNationEmbed(nations[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
