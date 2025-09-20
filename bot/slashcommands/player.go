package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PlayerCommand struct{}

func (cmd PlayerCommand) Name() string { return "player" }
func (cmd PlayerCommand) Description() string {
	return "Replies with information about a resident or townless player."
}

func (cmd PlayerCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about a player. Similar to /res in-game, but works for non-emc players.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("name", "The name of the player to query.", 3, 36),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "compare",
			Description: "Compare differences in statistics between two players.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("player-a", "The initial player name to compare with B.", 3, 36),
				discordutil.RequiredStringOption("player-b", "The secondary player name to compare with A.", 3, 36),
			},
		},
	}
}

func (cmd PlayerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	_, err = SendSinglePlayer(s, i.Interaction)
	return err
}

func SendSinglePlayer(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	data := i.ApplicationCommandData()
	playerNameArg := data.GetOption("query").GetOption("name").StringValue()

	players, err := oapi.QueryPlayers(strings.ToLower(playerNameArg))
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving player information :(")
	}

	if len(players) == 0 {
		return discordutil.FollowUpContent(s, i, fmt.Sprintf("No players retrieved. Player `%s` does not seem to exist.", playerNameArg))
	}

	embed := common.NewPlayerEmbed(players[0])
	return discordutil.FollowUpEmbeds(s, i, embed)
}
