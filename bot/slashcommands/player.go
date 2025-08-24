package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PlayerCommand struct{}

func (cmd PlayerCommand) Name() string { return "player" }
func (cmd PlayerCommand) Description() string {
	return "Replies with information about a resident or townless player."
}

func (cmd PlayerCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "name",
			Type:        discordgo.ApplicationCommandOptionString,
			Description: "The name of the player to retrieve information for.",
			Required:    true,
		},
	}
}

func (cmd PlayerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Defer the interaction immediately
	err := DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	_, err = SendSinglePlayer(s, i.Interaction)
	return err
}

func SendSinglePlayer(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	playerNameArg := i.ApplicationCommandData().GetOption("name").StringValue()

	players, err := oapi.QueryPlayers(strings.ToLower(playerNameArg))
	if err != nil {
		return FollowUpContent(s, i, "An error occurred retrieving player information :(")
	}

	if len(players) == 0 {
		return FollowUpContent(s, i, fmt.Sprintf("No players retrieved. Player `%s` does not seem to exist.", playerNameArg))
	}

	embed := common.CreatePlayerEmbed(players[0])
	return FollowUpEmbeds(s, i, embed)
}
