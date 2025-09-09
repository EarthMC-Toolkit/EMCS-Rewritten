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
	return "Retrieve information relating to one or more towns."
}

func (cmd TownCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "name",
			Type:        discordgo.ApplicationCommandOptionString,
			Description: "The name of the town to retrieve information for.",
			Required:    true,
		},
	}
}

func (cmd TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	_, err = SendSingleTown(s, i.Interaction)
	return err
}

func SendSingleTown(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	townNameArg := i.ApplicationCommandData().GetOption("name").StringValue()

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
