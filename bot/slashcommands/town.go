package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type TownCommand struct{}

func (cmd TownCommand) Name() string { return "town" }
func (cmd TownCommand) Description() string {
	return "Base command for town related subcommands."
}

func (cmd TownCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about a town. Similar to /t in-game.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("name", "The name of the town to query.", 2, 36),
			},
		},
	}
}

func (cmd TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		townNameArg := opt.GetOption("name").StringValue()
		_, err := executeQueryTown(s, i.Interaction, townNameArg)
		return err
	}

	return nil
}

func executeQueryTown(s *discordgo.Session, i *discordgo.Interaction, townName string) (*discordgo.Message, error) {
	var town *oapi.TownInfo

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		town, err = getTownFromOAPI(townName)
		if err != nil {
			return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("A database error occurred and the API failed during fallback!?```%s```", err))
		}
	} else {
		if len(townStore.Keys()) == 0 {
			return discordutil.FollowupContentEphemeral(s, i, "The town database is currently empty. This is unusual, but may resolve itself.")
		}

		town, _ = townStore.FindFirst(func(info oapi.TownInfo) bool {
			return strings.EqualFold(townName, info.Name) || townName == info.UUID
		})
	}

	if town == nil {
		return discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Town `%s` does not seem to exist.", townName))
	}

	embed := shared.NewTownEmbed(*town)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func getTownFromOAPI(townName string) (*oapi.TownInfo, error) {
	towns, err := oapi.QueryTowns(strings.ToLower(townName))
	if err != nil {
		return nil, err
	}

	if len(towns) == 0 {
		return nil, nil
	}

	t := towns[0]
	return &t, nil
}
