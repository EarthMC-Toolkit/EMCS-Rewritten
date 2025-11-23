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
	if err != nil || len(townStore.Keys()) < 1 {
		// fallback to oapi
		t, wasErr := getTownFromOAPI(townName)
		if wasErr {
			return discordutil.FollowupContent(s, i, "DB error occurred and OAPI error also occurred during fallback!?!")
		}

		town = t
	} else {
		t, err := townStore.FindFirst(func(townInfo oapi.TownInfo) bool {
			return strings.EqualFold(townName, townInfo.Name) || townName == townInfo.UUID
		})
		if err == nil {
			town = t // db success
		} else {
			// fallback to oapi
			oapiTown, wasErr := getTownFromOAPI(townName)
			if wasErr {
				return discordutil.FollowupContent(s, i, "Town does not exist in the database and the OAPI failed!")
			}

			town = oapiTown
		}
	}

	// DB and OAPI are working normally but town wasn't found in either.
	if town == nil {
		return discordutil.FollowupContent(s, i, fmt.Sprintf("Town `%s` does not seem to exist.", townName))
	}

	embed := shared.NewTownEmbed(*town)
	return discordutil.FollowupEmbeds(s, i, embed)
}

func getTownFromOAPI(townName string) (*oapi.TownInfo, bool) {
	towns, err := oapi.QueryTowns(strings.ToLower(townName))
	if err != nil {
		return nil, true
	}

	if len(towns) == 0 {
		return nil, false
	}

	t := towns[0]
	return &t, false
}
