package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/utils/discordutil"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type MysteryMasterCommand struct{}

func (cmd MysteryMasterCommand) Name() string { return "mysterymaster" }
func (cmd MysteryMasterCommand) Description() string {
	return "Top 50 players in Mystery Master. Change = movement on the scoreboard."
}

func (cmd MysteryMasterCommand) Options() AppCommandOpts {
	return nil
}

func (cmd MysteryMasterCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	_, err := SendMysteryMasterList(s, i.Interaction)
	return err
}

func SendMysteryMasterList(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	mmList, err := oapi.QueryMysteryMaster()
	if err != nil {
		return discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving town information :(",
		})
	}

	count := len(mmList)
	if count == 0 {
		return discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No mystery masters seem to exist? The API may be broken.",
		})
	}

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 15
	paginator := discordutil.NewInteractionPaginator(s, i, count, perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		content := ""
		for idx, item := range mmList[start:end] {
			changeEmoji := lo.Ternary(*item.Change == "UP", common.EMOJIS.ARROW_UP_GREEN, common.EMOJIS.ARROW_DOWN_RED)
			content += fmt.Sprintf("%d. %s %s\n", start+idx+1, changeEmoji, item.Name) // add start to keep index across pages. just idx+1 shows 1-perPage every time.
		}

		data.Content = content + fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages())
		data.Components = []discordgo.MessageComponent{
			paginator.NewNavigationButtonRow(),
		}
	}

	return nil, paginator.Start()
}
