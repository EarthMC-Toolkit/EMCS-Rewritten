package slashcommands

import (
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type ServerCommand struct{}

func (cmd ServerCommand) Name() string { return "server" }
func (cmd ServerCommand) Description() string {
	return "Base command for server related subcommands."
}

func (cmd ServerCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "info",
			Description: "Replies with information about the server (time, new day, vp).",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "player-stats",
			Description: "Sends a paginated list with the server's all-time player statistics.",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "sort",
					Description: "Sort order of the keys used before displaying the list.",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Grouped", Value: "grouped"},
						{Name: "Alphabetical", Value: "alphabetical"},
					},
				},
			},
		},
	}
}

func (cmd ServerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("info"); opt != nil {
		_, err := executeServerInfo(s, i.Interaction)
		return err
	}
	if opt := cdata.GetOption("player-stats"); opt != nil {
		sortArg := opt.GetOption("sort").StringValue()
		_, err := executeServerPlayerStats(s, i.Interaction, sortArg)
		return err
	}

	return nil
}

func executeServerInfo(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	serverStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.SERVER_STORE)
	if err != nil {
		return nil, err
	}

	info, err := serverStore.GetKey("info")
	if err != nil {
		log.Printf("failed to get serverinfo from db:\n%v", err)
		return discordutil.FollowupContentEphemeral(s, i, "An error occurred retrieving server info from the database. Check the console.")
	}

	timePassed := info.Stats.Time
	serverTime := info.Timestamps.ServerTimeOfDay
	newDayTime := info.Timestamps.NewDayTime

	timestampsField := &discordgo.MessageEmbedField{
		Name: "Timestamps",
		Value: strings.Join([]string{
			fmt.Sprintf("Server Time Of Day: `%d`", serverTime),
			fmt.Sprintf("Time Passed In Current Day: `%d`", timePassed),
			fmt.Sprintf("New Day At: `%d`", newDayTime),
		}, "\n"),
		Inline: true,
	}

	vpTarget := info.VoteParty.Target
	vpRemaining := info.VoteParty.NumRemaining
	vpField := &discordgo.MessageEmbedField{
		Name:   "Vote Party",
		Value:  utils.HumanizedSprintf("Votes Completed/Target: `%d`/`%d`\nVotes Remaining: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
		Inline: true,
	}

	statsField := &discordgo.MessageEmbedField{
		Name: "Statistics",
		Value: strings.Join([]string{
			utils.HumanizedSprintf("Online: `%d`", info.Stats.NumOnlinePlayers),
			utils.HumanizedSprintf("Townless (Online/Total): `%d`/`%d`", info.Stats.NumOnlineNomads, info.Stats.NumNomads),
			utils.HumanizedSprintf("Residents: `%d`", info.Stats.NumResidents),
			utils.HumanizedSprintf("Towns: `%d`", info.Stats.NumTowns),
			utils.HumanizedSprintf("Nations: `%d`", info.Stats.NumNations),
			utils.HumanizedSprintf("Quarters: `%d`", info.Stats.NumQuarters),
		}, "\n"),
		Inline: false,
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Server Info",
		Fields: []*discordgo.MessageEmbedField{timestampsField, vpField, statsField},
		Color:  discordutil.BLURPLE,
		Footer: shared.DEFAULT_FOOTER,
	}

	return discordutil.FollowupEmbeds(s, i, embed)
}

// TODO:
// Every time this cmd is run, we should attempt to cache stats if the last cache was more than 30s ago.
// The cache will then be used for subsequent cmds until next expiry to reduce API calls that contribute to rate limit unnecessarily.
func executeServerPlayerStats(s *discordgo.Session, i *discordgo.Interaction, sortArg string) (*discordgo.Message, error) {
	fmt.Printf("Sort arg: '%s'", sortArg)
	return nil, nil
}

// func executeServerTownyStats(s *discordgo.Session, i *discordgo.Interaction, sortArg string) (*discordgo.Message, error) {
// 	return nil, nil
// }
