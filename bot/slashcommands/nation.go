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
	return "Retrieve information relating to one or more nations."
}

func (cmd NationCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "name",
			Type:        discordgo.ApplicationCommandOptionString,
			Description: "The name of the nation to retrieve information for.",
			Required:    true,
		},
	}
}

func (cmd NationCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
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
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving nation information :(")
	}

	if len(nations) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No nations retrieved. Nation `%s` does not seem to exist.", nationNameArg))
	}

	embed := common.CreateNationEmbed(nations[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
