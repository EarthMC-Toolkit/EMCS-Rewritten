package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
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

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		townNameArg := opt.GetOption("name").StringValue()
		_, err := executeQueryTown(s, i.Interaction, townNameArg)
		return err
	}

	return nil
}

func executeQueryTown(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	towns, err := oapi.QueryTowns(strings.ToLower(townName))
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving town information :(")
	}

	if len(towns) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No towns retrieved. Town `%s` does not seem to exist.", townName))
	}

	embed := common.NewTownEmbed(towns[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
