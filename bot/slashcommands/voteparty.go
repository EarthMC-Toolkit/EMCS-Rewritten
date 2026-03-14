package slashcommands

import (
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
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
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	serverStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.SERVER_STORE)
	if err != nil {
		return err
	}

	info, err := serverStore.Get("info")
	if err != nil {
		err := fmt.Errorf("failed to execute /serverinfo. could not find 'info' key in 'server' store")
		discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: err.Error(),
		})

		return err
	}

	vpTarget := info.VoteParty.Target
	vpRemaining := info.VoteParty.NumRemaining
	embed := &discordgo.MessageEmbed{
		Title: "VoteParty Status",
		Description: utils.HumanizedSprintf(
			"Votes Completed/Target: `%d`/`%d`\n\nThere are `%d` votes remaining until the next VoteParty occurs.",
			vpTarget-vpRemaining, vpTarget, vpRemaining,
		),
		Color:  discordutil.BLURPLE,
		Footer: embeds.DEFAULT_FOOTER,
	}

	_, err = discordutil.FollowupEmbeds(s, i.Interaction, embed)
	return err
}
