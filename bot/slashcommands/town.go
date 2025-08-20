package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/oapi"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var NameOption = &discordgo.ApplicationCommandOption{
	Name:        "name",
	Type:        discordgo.ApplicationCommandOptionString,
	Description: "The name of the town to retrieve information for.",
	Required:    true,
}

type TownCommand struct{}

func (TownCommand) Name() string                           { return "town" }
func (TownCommand) Description() string                    { return "Retrieve information relating to one or more towns." }
func (TownCommand) Type() discordgo.ApplicationCommandType { return discordgo.ChatApplicationCommand }
func (TownCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		NameOption,
	}
}

func (TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Defer the interaction immediately
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: 5,
	})
	if err != nil {
		return err
	}

	townNameArg := i.ApplicationCommandData().GetOption("name").StringValue()

	towns, err := oapi.QueryTowns(strings.ToLower(townNameArg))
	if err != nil {
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "An error occurred retrieving town information :(",
		})

		return err
	}

	if len(towns) == 0 {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("No towns retrieved. Town `%s` does not seem to exist.", townNameArg),
		})

		return err
	}

	embed := common.CreateTownEmbed(i.Interaction, towns[0])
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})

	return err
}
