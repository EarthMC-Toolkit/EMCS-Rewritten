package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/discordutil"

	"github.com/bwmarrin/discordgo"
)

type MysteryMasterCommand struct{}

func (cmd MysteryMasterCommand) Name() string { return "serverinfo" }
func (cmd MysteryMasterCommand) Description() string {
	return "Retrieve top 50 players participating in Mystery Master. Change = direction moved on the score board."
}

func (cmd MysteryMasterCommand) Options() []*discordgo.ApplicationCommandOption {
	return nil
}

func (cmd MysteryMasterCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	_, err := SendMysteryMasterList(s, i.Interaction)
	return err
}

func SendMysteryMasterList(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	list, err := oapi.QueryMysteryMaster()
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred retrieving town information :(")
	}

	if len(list) == 0 {
		return discordutil.FollowUpContent(s, i, "No mystery masters seem to exist? The API may be broken.")
	}

	// do paginator

	return nil, nil
}
