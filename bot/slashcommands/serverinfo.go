package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/discordutil"
	"emcsrw/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type ServerInfoCommand struct{}

func (cmd ServerInfoCommand) Name() string { return "serverinfo" }
func (cmd ServerInfoCommand) Description() string {
	return "Replies with information about the server and voteparty status."
}

func (cmd ServerInfoCommand) Options() []*discordgo.ApplicationCommandOption {
	return nil
}

func (cmd ServerInfoCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	info, err := database.GetInsensitive[oapi.ServerInfo](db, "serverinfo")
	if err != nil {
		fmt.Printf("failed to get serverinfo from db:\n%v", err)
		return discordutil.Reply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving server info from the database. check the console",
		})
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
		Inline: true,
	}

	vpTarget := info.VoteParty.Target
	vpRemaining := info.VoteParty.NumRemaining
	vpField := &discordgo.MessageEmbedField{
		Name:   "Vote Party",
		Value:  utils.HumanizedSprintf("Votes Completed/Target: `%d`/`%d`\nVotes Left: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
		Inline: true,
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Server Info",
		Fields: []*discordgo.MessageEmbedField{statsField, vpField},
		Color:  discordutil.BLURPLE,
		Footer: common.DEFAULT_FOOTER,
	}

	// Should generally respond in 3 seconds. May need to defer in future?
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
