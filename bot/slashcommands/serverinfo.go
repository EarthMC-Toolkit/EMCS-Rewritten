package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type ServerInfoCommand struct{}

func (cmd ServerInfoCommand) Name() string { return "serverinfo" }
func (cmd ServerInfoCommand) Description() string {
	return "Replies with information about the server and voteparty status."
}

func (cmd ServerInfoCommand) Options() AppCommandOpts {
	return nil
}

func (cmd ServerInfoCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	db := store.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	info, err := store.GetInsensitive[oapi.ServerInfo](db, "serverinfo")
	if err != nil {
		log.Printf("failed to get serverinfo from db:\n%v", err)
		return discordutil.SendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving server info from the database. Check the console.",
		})
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
		Value:  utils.HumanizedSprintf("Votes Completed/Target: `%d`/`%d`\nVotes Left: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
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
		Footer: common.DEFAULT_FOOTER,
	}

	// Should generally respond in 3 seconds. May need to defer in future?
	return discordutil.SendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}
