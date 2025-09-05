package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/discordutil"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type MysteryMasterCommand struct{}

func (cmd MysteryMasterCommand) Name() string { return "mysterymaster" }
func (cmd MysteryMasterCommand) Description() string {
	return "Top 50 players in Mystery Master. Change = movement on the scoreboard."
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
	paginator.PageFunc = func(curPage, perPage int, data *discordgo.InteractionResponseData) {
		start := curPage * perPage
		end := min(start+perPage, count)

		content := fmt.Sprintf("Page %d/%d\n\n", curPage+1, paginator.TotalPages())
		for idx, item := range list[start:end] {
			content += fmt.Sprintf("%d. %s - %s\n", start+idx+1, item.Name, *item.Change)
		}

		fmt.Println("curPage:", curPage, "start:", start, "end:", end, "count:", count)

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
