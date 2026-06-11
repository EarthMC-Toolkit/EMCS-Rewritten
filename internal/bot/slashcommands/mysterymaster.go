package slashcommands

import (
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/discordutil"
	"fmt"
	"strings"

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
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	_, err := SendMysteryMasterList(s, i.Interaction)
	return err
}

func SendMysteryMasterList(s *discordgo.Session, i *discordgo.Interaction) (*discordgo.Message, error) {
	mmList, err := oapi.QueryMysteryMaster().Execute()
	if err != nil {
		return discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving town information :(",
		})
	}

	count := len(mmList)
	if count == 0 {
		return discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
			Content: "No mystery masters seem to exist? The API may be broken.",
		})
	}

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 15
	paginator := discordutil.NewInteractionPaginator(s, i, count, perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		content := strings.Builder{}
		for idx, item := range mmList[start:end] {
			changeEmoji := lo.Ternary(*item.Change == "UP", shared.EMOJIS.ARROW_UP_GREEN, shared.EMOJIS.ARROW_DOWN_RED)

			listIdx := start + idx + 1 // add start to keep index across pages. idx+1 would just show 1 to <perPage> every time.
			fmt.Fprintf(&content, "%d. %s %s\n", listIdx, changeEmoji, item.Name)
		}

		data.Content = content.String() + fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages())
	}

	return nil, paginator.Start()
}
