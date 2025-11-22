package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
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

	townsStore, err := database.GetStoreForMap[oapi.TownInfo](shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		log.Printf("failed to get towns from db:\n%v", err)
		_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving towns from the database. Check the console.",
		})

		return err
	}

	towns := townsStore.Entries()
	ruined := lo.FilterMapToSlice(towns, func(_ string, t oapi.TownInfo) (oapi.TownInfo, bool) {
		return t, t.Status.Ruined
	})

	sort.Slice(ruined, func(i, j int) bool {
		return *ruined[i].Timestamps.RuinedAt < *ruined[j].Timestamps.RuinedAt
	})

	count := len(ruined)
	perPage := 10

	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, len(ruined), perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		desc := ""
		items := ruined[start:end]
		count := len(items)

		for idx, t := range items {
			ruinedTs := *t.Timestamps.RuinedAt
			ruinedTime := time.UnixMilli(int64(*t.Timestamps.RuinedAt))
			after72h := ruinedTime.Add(72 * time.Hour)

			nextNewDay := time.Date(after72h.Year(), after72h.Month(), after72h.Day(), 11, 0, 0, 0, time.UTC)
			if !nextNewDay.After(after72h) {
				nextNewDay = nextNewDay.Add(24 * time.Hour)
			}

			chunks := utils.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := utils.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

			spawn := t.Coordinates.Spawn
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)", spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z)

			desc += fmt.Sprintf(
				"%d. **%s** fell into ruin <t:%d:R> at %s. %sG %s\nDeletion on `%s` (<t:%d:R>).\n\n",
				start+idx+1, t.Name, ruinedTs/1000, locationLink, balance, chunks, utils.FormatTime(nextNewDay), nextNewDay.Unix(),
			)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("List of Ruined Towns [%d]", count),
			Footer:      shared.DEFAULT_FOOTER,
			Description: desc + fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages()),
			Color:       discordutil.DARK_GOLD,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
}
