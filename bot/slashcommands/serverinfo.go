package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/utils"
	"fmt"

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
	serverInfo, err := oapi.QueryServer()
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error fetching server info: %v", err),
			},
		})
	}

	onlineStr := utils.HumanizedSprintf("Online: `%d`\n", serverInfo.Stats.NumOnlinePlayers)
	townlessStr := utils.HumanizedSprintf("Townless (Online/Total): `%d`/`%d`\n", serverInfo.Stats.NumOnlineNomads, serverInfo.Stats.NumNomads)
	residentsStr := utils.HumanizedSprintf("Residents: `%d`\n", serverInfo.Stats.NumResidents)
	townsStr := utils.HumanizedSprintf("Towns: `%d`\n", serverInfo.Stats.NumTowns)
	nationsStr := utils.HumanizedSprintf("Nations: `%d`", serverInfo.Stats.NumNations)
	statsField := &discordgo.MessageEmbedField{
		Name:   "Statistics",
		Value:  onlineStr + townlessStr + residentsStr + townsStr + nationsStr,
		Inline: true,
	}

	vpTarget := serverInfo.VoteParty.Target
	vpRemaining := serverInfo.VoteParty.NumRemaining
	vpField := &discordgo.MessageEmbedField{
		Name:   "Vote Party",
		Value:  utils.HumanizedSprintf("Current/Target: `%d`/`%d`\nRemaining: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
		Inline: true,
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Server Info",
		Fields: []*discordgo.MessageEmbedField{statsField, vpField},
		Color:  0x5C4DFF,
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
