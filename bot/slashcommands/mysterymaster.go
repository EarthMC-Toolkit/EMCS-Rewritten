package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/discordutil"
	"fmt"

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

	count := len(list)
	if count == 0 {
		return discordutil.FollowUpContent(s, i, "No mystery masters seem to exist? The API may be broken.")
	}

	// Init paginator with 20 items per page. Pressing a btn will change the current page and call PageFunc again.
	paginator := discordutil.NewInteractionPaginator(s, i, count, 20)
	paginator.PageFunc = func(curPage, pageLen int, data *discordgo.InteractionResponseData) {
		start := curPage * pageLen
		end := min(start+pageLen, count)

		content := fmt.Sprintf("Page %d/%d\n\n", curPage+1, paginator.TotalPages())
		for i, item := range list[start:end] {
			content += fmt.Sprintf("%d. %s - %s\n", i+1, item.Name, *item.Change)
		}

		data.Content = content
		data.Components = []discordgo.MessageComponent{
			paginator.NewNavigationButtonRow(),
		}
	}

	err = paginator.Start()
	if err != nil {
		return discordutil.FollowUpContent(s, i, "An error occurred starting embed pagination.")
	}

	return nil, nil
}
