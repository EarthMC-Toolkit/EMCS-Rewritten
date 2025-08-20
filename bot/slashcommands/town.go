package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/oapi"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var townNameOption = &discordgo.ApplicationCommandOption{
	Name:        "name",
	Type:        discordgo.ApplicationCommandOptionString,
	Description: "The name of the town to retrieve information for.",
	Required:    true,
}

type TownCommand struct{}

func (TownCommand) Name() string        { return "town" }
func (TownCommand) Description() string { return "Retrieve information relating to one or more towns." }

func (TownCommand) Type() discordgo.ApplicationCommandType {
	return discordgo.ChatApplicationCommand
}

func (TownCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		townNameOption,
	}
}

func (TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Defer the interaction immediately
	err := DeferReply(s, i.Interaction)
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
		return FollowUpContent(s, i, "An error occurred retrieving town information :(")
	}

	if len(towns) == 0 {
		return FollowUpContent(s, i, fmt.Sprintf("No towns retrieved. Town `%s` does not seem to exist.", townNameArg))
	}

	embed := common.CreateTownEmbed(towns[0])
	return FollowUpEmbeds(s, i, embed)
}
