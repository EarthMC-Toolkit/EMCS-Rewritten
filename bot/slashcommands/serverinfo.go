package slashcommands

import (
	"emcsrw/oapi"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(ServerInfo{})
}

type ServerInfo struct{}

func (ServerInfo) Name() string                           { return "serverinfo" }
func (ServerInfo) Description() string                    { return "Replies with information about the server" }
func (ServerInfo) Type() discordgo.ApplicationCommandType { return discordgo.ChatApplicationCommand }
func (ServerInfo) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{}
}

func (ServerInfo) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	serverInfo, err := oapi.ServerInfo()
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error fetching server info: %v", err),
			},
		})
	}

	townlessStr := fmt.Sprintf("Townless (Online/Total): %d/%d\n", serverInfo.Stats.NumOnlineNomads, serverInfo.Stats.NumNomads)
	onlineStr := fmt.Sprintf("Online: %d\n", serverInfo.Stats.NumOnlinePlayers)
	residentsStr := fmt.Sprintf("Residents: %d\n", serverInfo.Stats.NumResidents)
	townsStr := fmt.Sprintf("Towns: %d\n", serverInfo.Stats.NumTowns)
	nationsStr := fmt.Sprintf("Nations: %d", serverInfo.Stats.NumNations)
	statsField := &discordgo.MessageEmbedField{
		Name:   "Statistics",
		Value:  onlineStr + townlessStr + residentsStr + townsStr + nationsStr,
		Inline: true,
	}

	vpTarget := serverInfo.VoteParty.Target
	vpRemaining := serverInfo.VoteParty.NumRemaining
	vpField := &discordgo.MessageEmbedField{
		Name:   "Vote Party",
		Value:  fmt.Sprintf("Target: %d\nRemaining: %d\nRequired: %d", vpTarget, vpRemaining, vpTarget-vpRemaining),
		Inline: true,
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Server Info",
		Fields: []*discordgo.MessageEmbedField{statsField, vpField},
		Color:  0x5C4DFF,
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
