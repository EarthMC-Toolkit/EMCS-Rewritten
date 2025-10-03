package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"

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
	db, err := store.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	if err != nil {
		return err
	}

	serverStore, err := store.GetStore[oapi.ServerInfo](db, "server")
	if err != nil {
		return err
	}

	info, err := serverStore.GetKey("info")
	if err != nil {
		err := fmt.Errorf("failed to execute /serverinfo. could not find 'info' key in 'server' store")
		discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: err.Error(),
		})

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
