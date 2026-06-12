package slashcommands

import (
	"emcsrw/internal/shared"
	"emcsrw/pkg/api"
	"emcsrw/pkg/api/mapi"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
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

func (cmd VisibleCommand) Options() []AppCommandOpt {
	return nil
}

func (cmd VisibleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	oapiPlayers, visible, err := api.QueryVisiblePlayers()
	if err != nil {
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred during the map request or response parsing :(",
		})

		return err
	}

	playerLookup := make(map[string]oapi.PlayerInfo, len(oapiPlayers))
	for _, p := range oapiPlayers {
		playerLookup[p.Name] = p
	}

	// sort alphabetically
	utils.KeySort(visible, []utils.KeySortOption[mapi.MapPlayer]{
		{Compare: func(a, b mapi.MapPlayer) bool { return a.Name < b.Name }}, // ascending (A-Z)
	})

	count := len(visible)
	if count == 0 {
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred. Players array is empty (server may be partially down).",
		})

		return err
	}

	perPage := 10
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, count, perPage).
		WithTimeout(30 * time.Second)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		b := strings.Builder{} // build the description faster than regular Sprintf concat
		for idx, p := range visible[start:end] {
			pinfo := playerLookup[p.Name]

			townName := "Townless"
			if pinfo.Town.Name != nil {
				townName = *pinfo.Town.Name
			}

			locStr := fmt.Sprintf("[%d, %d, %d](https://map.earthmc.net/?x=%d&z=%d&zoom=4)", p.X, p.Y, p.Z, p.X, p.Z)
			balStr := fmt.Sprintf("`%0.0f`G %s", pinfo.Bal(), shared.EMOJIS.GOLD_INGOT)

			registeredStr := fmt.Sprintf("<t:%d:R>", pinfo.Timestamps.Registered/1000)

			joinedTownStr := "N/A"
			if pinfo.Timestamps.JoinedTownAt != nil {
				joinedTownStr = fmt.Sprintf("<t:%d:R>", *pinfo.Timestamps.JoinedTownAt/1000)
			}

			fmt.Fprintf(&b, "%d. **%s** (%s) | %s | %s\nRank: `%s` | Registered: %s | Joined Town: %s\n\n",
				start+idx+1, p.Name, townName, locStr, balStr,
				pinfo.RankOrRole(), registeredStr, joinedTownStr,
			)
		}

		title := fmt.Sprintf("List of Visible Players [%d]", count)
		desc := b.String() + fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())

		embed := discordutil.NewEmbedBuilder(&discordutil.BLURPLE, &title, &desc, nil)
		data.Embeds = append(data.Embeds, embed.Build())
	}

	return paginator.Start()
}
