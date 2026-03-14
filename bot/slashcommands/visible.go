package slashcommands

import (
	"emcsrw/api/mapi"
	"emcsrw/shared/embeds"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type VisibleCommand struct{}

func (cmd VisibleCommand) Name() string { return "visible" }
func (cmd VisibleCommand) Description() string {
	return "Shows the list of online players that are not currently under a block."
}

func (cmd VisibleCommand) Options() AppCommandOpts {
	return nil
}

func (cmd VisibleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	visible, err := mapi.GetVisiblePlayers()
	if err != nil {
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred during the map request or response parsing :(",
		})

		return err
	}

	count := len(visible)
	if count == 0 {
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred. Players array is empty (server may be partially down).",
		})

		return err
	}

	perPage := 20
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, count, perPage).
		WithTimeout(30 * time.Second)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		desc := strings.Builder{}
		for idx, p := range visible[start:end] {
			loc := fmt.Sprintf("%d, %d, %d", p.X, p.Y, p.Z)
			fmt.Fprintf(&desc, "%d. **%s** | `%s`\n", start+idx+1, p.Name, loc)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("List of Visible Players [%d]", count),
			Footer:      embeds.DEFAULT_FOOTER,
			Description: desc.String() + fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages()),
		}

		data.Embeds = append(data.Embeds, embed)
	}

	return paginator.Start()
}
