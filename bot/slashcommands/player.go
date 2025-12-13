package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
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
		mdb, err := database.Get(shared.ACTIVE_MAP)
		if err != nil {
			return discordutil.FollowupContent(s, i, "An error occurred getting the database for map: "+shared.ACTIVE_MAP)
		}

		playerStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
		if err != nil {
			return discordutil.FollowupContent(s, i, "An error occurred retrieving player information :(")
		}

		// check if they opted out (massive pussy) or actually don't exist.
		p, err := playerStore.FindFirst(func(p database.BasicPlayer) bool {
			return strings.EqualFold(p.Name, playerName)
		})
		if err != nil {
			content := fmt.Sprintf("Player `%s` could not be retrieved from the EarthMC API.", playerName)
			content += "\nIt is possible that this player is both townless and has opted-out."
			return discordutil.FollowupContent(s, i, content)
		}

		embed := embeds.NewBasicPlayerEmbed(*p)
		return discordutil.FollowupEmbeds(s, i, embed)
	}

	embed := embeds.NewPlayerEmbed(players[0])
	return discordutil.FollowupEmbeds(s, i, embed)
}
