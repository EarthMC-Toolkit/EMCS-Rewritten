package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/discordutil"
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
				discordutil.RequiredStringOption("name", "The name of the town to query.", 2, 36),
			},
		},
	}
}

func (cmd TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	_, err = QueryTown(s, i.Interaction)
	return err
}

func QueryTown(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	data := i.ApplicationCommandData()
	townNameArg := data.GetOption("query").GetOption("name").StringValue()

	towns, err := oapi.QueryTowns(strings.ToLower(townNameArg))
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving town information :(")
	}

	if len(towns) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No towns retrieved. Town `%s` does not seem to exist.", townNameArg))
	}

	embed := common.NewTownEmbed(towns[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
