package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
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

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		playerNameArg := opt.GetOption("name").StringValue()
		_, err = executeQueryPlayer(s, i.Interaction, playerNameArg)
		return err
	}

	return nil
}

func executeQueryPlayer(s *discordgo.Session, i *discordgo.Interaction, playerName string) (*discordgo.Message, error) {
	players, err := oapi.QueryPlayers(strings.ToLower(playerName))
	if err != nil {
		return discordutil.FollowupContent(s, i, "An error occurred retrieving player information :(")
	}

	if len(players) == 0 {
		return discordutil.FollowupContent(s, i, fmt.Sprintf("No players retrieved. Player `%s` does not seem to exist.", playerName))
	}

	embed := shared.NewPlayerEmbed(players[0])
	return discordutil.FollowupEmbeds(s, i, embed)
}
