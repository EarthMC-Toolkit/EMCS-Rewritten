package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/discordutil"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PlayerCommand struct{}

func (cmd PlayerCommand) Name() string { return "player" }
func (cmd PlayerCommand) Description() string {
	return "Replies with information about a resident or townless player."
}

func (cmd PlayerCommand) Options() []AppCommandOpt {
	return []AppCommandOpt{
		discordutil.SubcommandOption("query", "Query information about a player. Similar to /res in-game, but works for non-emc players.",
			discordutil.RequiredStringOption("name", "The name of the player to query.", 3, 36),
		),
		// discordutil.SubcommandOption("compare", "Compare differences in statistics between two players.",
		// 	discordutil.RequiredStringOption("player-a", "The initial player name to compare with B.", 3, 36),
		// 	discordutil.RequiredStringOption("player-b", "The secondary player name to compare with A.", 3, 36),
		// ),
	}
}

func (cmd PlayerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		playerNameArg := opt.GetOption("name").StringValue()
		_, err := executeQueryPlayer(s, i.Interaction, playerNameArg)
		return err
	}
	if opt := cdata.GetOption("compare"); opt != nil {
		if err := discordutil.DeferReply(s, i.Interaction); err != nil {
			return err
		}

		discordutil.ReplyWithError(s, i.Interaction, errors.New("Command not implemented yet."))
		return nil
	}

	return nil
}

func executeQueryPlayer(s *discordgo.Session, i *discordgo.Interaction, playerName string) (*discordgo.Message, error) {
	msg := discordutil.NewMessageBuilder()

	tokens := oapi.Dispatcher.GetBucketTokens()
	if len(tokens) < 1 {
		msg.SetContent(shared.EMOJIS.LOADING + " No tokens available to query the API. Queuing your request..")
		discordutil.SendReply(s, i, msg.InteractionData())
		msg.SetContent("")
	} else {
		discordutil.DeferReply(s, i)
	}

	players, apiErr := oapi.QueryPlayers(strings.ToLower(playerName)).Execute()
	if apiErr != nil {
		desc := ":warning: The EarthMC API is likely down right now. As such, some data may be missing until it is online again."
		embed, err := buildBasicPlayerEmbed(playerName, desc)
		if err != nil {
			content := fmt.Sprintf("An error occurred retrieving player information :(```%s```", apiErr)
			msg.SetContent(content)
		} else {
			msg.SetEmbeds(embed)
		}

		return discordutil.EditReply(s, i, msg.InteractionData())
	}

	// API is up, but no results.
	if len(players) == 0 {
		desc := ":warning: Limited data available. This player has chosen to opt out of the EarthMC API, what a *pussy*."
		embed, err := buildBasicPlayerEmbed(playerName, desc)
		if err != nil {
			content := fmt.Sprintf("Player `%s` could not be retrieved from the EarthMC API.", playerName)
			content += "\nIt is possible that this player is both townless and has opted-out."

			msg.SetContent(content)
		} else {
			msg.SetEmbeds(embed)
		}

		return discordutil.EditReply(s, i, msg.InteractionData())
	}

	// API is up and results received, everything working normally.
	msg.SetEmbeds(shared.NewPlayerEmbed(s, players[0]))
	return discordutil.EditReply(s, i, msg.InteractionData())
}

func buildBasicPlayerEmbed(playerName string, desc string) (*discordgo.MessageEmbed, error) {
	playerStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.PLAYERS_STORE)
	if err != nil {
		return nil, fmt.Errorf("error querying player `%s`: failed to get player store from DB", playerName)
	}

	// check if they opted out (massive pussy) or actually don't exist.
	p, err := playerStore.Find(func(p database.BasicPlayer) bool {
		return strings.EqualFold(p.Name, playerName)
	})
	if err != nil {
		return nil, fmt.Errorf("error querying player `%s`: opted-out or does not exist", playerName)
	}

	return shared.NewBasicPlayerEmbed(*p, desc), nil
}
