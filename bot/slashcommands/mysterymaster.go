package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/discordutil"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
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

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	paginator := discordutil.NewInteractionPaginator(s, i, count, 15)
	paginator.PageFunc = func(curPage, perPage int, data *discordgo.InteractionResponseData) {
		start := curPage * perPage
		end := min(start+perPage, count)

		var content string
		for idx, item := range list[start:end] {
			changeEmoji := lo.Ternary(*item.Change == "UP", common.EMOJIS.ARROW_UP_GREEN, common.EMOJIS.ARROW_DOWN_RED)
			content += fmt.Sprintf("%d. %s %s\n", start+idx+1, changeEmoji, item.Name) // add start to keep index across pages. just idx+1 shows 1-perPage every time.
		}

		data.Content = content + fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages())
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
