package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/oapi"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type NationCommand struct{}

func (NationCommand) Name() string { return "nation" }
func (NationCommand) Description() string {
	return "Retrieve information relating to one or more nations."
}

func (NationCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "name",
			Type:        discordgo.ApplicationCommandOptionString,
			Description: "The name of the nation to retrieve information for.",
			Required:    true,
		},
	}
}

func (NationCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Defer the interaction immediately
	err := DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	_, err = SendSingleNation(s, i.Interaction)
	return err
}

func SendSingleNation(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	nationNameArg := i.ApplicationCommandData().GetOption("name").StringValue()
	nations, err := oapi.QueryNations(strings.ToLower(nationNameArg))
	if err != nil {
		return FollowUpContent(s, i, "An error occurred retrieving town information :(")
	}

	embed := common.CreateNationEmbed(nations[0])
	return FollowUpEmbeds(s, i, embed)
}
