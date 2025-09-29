package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type RuinedCommand struct{}

func (cmd RuinedCommand) Name() string { return "ruined" }
func (cmd RuinedCommand) Description() string {
	return "Retrieves a list of all ruined towns with their time of ruin."
}

func (cmd RuinedCommand) Options() AppCommandOpts {
	return nil
}

func (cmd RuinedCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	db := store.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	towns, err := store.GetInsensitive[[]oapi.TownInfo](db, "towns")
	if err != nil {
		log.Printf("failed to get towns from db:\n%v", err)
		_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving towns from the database. Check the console.",
		})

		return err
	}

	ruined := lo.FilterMap(*towns, func(t oapi.TownInfo, _ int) (oapi.TownInfo, bool) {
		return t, t.Status.Ruined
	})

	sort.Slice(ruined, func(i, j int) bool {
		return *ruined[i].Timestamps.RuinedAt < *ruined[j].Timestamps.RuinedAt
	})

	count := len(ruined)
	perPage := 10

	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, len(ruined), perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		desc := ""
		items := ruined[start:end]
		count := len(items)

		for idx, t := range items {
			ruinedTs := *t.Timestamps.RuinedAt
			deleteTs := time.UnixMilli(int64(ruinedTs)).Add(74 * time.Hour) // 72 UTC but EMC goes on UTC+2 (i think?)

			chunks := utils.HumanizedSprintf("%s `%d`", common.EMOJIS.CHUNK, t.Stats.NumTownBlocks)
			balance := utils.HumanizedSprintf("%s `%0.0f`", common.EMOJIS.GOLD_INGOT, t.Stats.Balance)

			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			desc += fmt.Sprintf(
				"%d. **%s** fell into ruin <t:%d:R> | %s Chunks %sG\nScheduled for deletion in <t:%d:R>. Located at %s\n\n",
				start+idx+1, t.Name, ruinedTs/1000, chunks, balance, deleteTs.Unix(), locationLink,
			)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("List of Ruined Towns [%d]", count),
			Footer:      common.DEFAULT_FOOTER,
			Description: desc + fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages()),
			Color:       discordutil.DARK_GOLD,
		}

		data.Embeds = append(data.Embeds, embed)
		data.Components = []discordgo.MessageComponent{
			paginator.NewNavigationButtonRow(),
		}
	}

	return paginator.Start()
}
