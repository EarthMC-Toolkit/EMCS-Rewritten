package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/discordutil"
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

	_, err = QueryNation(s, i.Interaction)
	return err
}

func QueryNation(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	data := i.ApplicationCommandData()
	nationNameArg := data.GetOption("query").GetOption("name").StringValue()

	nations, err := oapi.QueryNations(strings.ToLower(nationNameArg))
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving nation information :(")
	}

	if len(nations) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No nations retrieved. Nation `%s` does not seem to exist.", nationNameArg))
	}

	embed := common.NewNationEmbed(nations[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
