package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"

	"github.com/bwmarrin/discordgo"
)

type VotePartyCommand struct{}

func (cmd VotePartyCommand) Name() string { return "vp" }
func (cmd VotePartyCommand) Description() string {
	return "Retrieves info on the current status of the VoteParty."
}

func (cmd VotePartyCommand) Options() AppCommandOpts {
	return nil
}

func (cmd VotePartyCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	info, err := database.GetInsensitive[oapi.ServerInfo](db, "serverinfo")
	if err != nil {
		// TODO: Respond to interaction with appropriate err msg
		return err
	}

	vpTarget := info.VoteParty.Target
	vpRemaining := info.VoteParty.NumRemaining
	embed := &discordgo.MessageEmbed{
		Title:       "VoteParty Status",
		Description: utils.HumanizedSprintf("Votes Completed/Target: `%d`/`%d`\nVotes Left: `%d`", vpTarget-vpRemaining, vpTarget, vpRemaining),
		Color:       discordutil.BLURPLE,
		Footer:      common.DEFAULT_FOOTER,
	}

	// Should generally respond in 3 seconds. May need to defer in future?
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
