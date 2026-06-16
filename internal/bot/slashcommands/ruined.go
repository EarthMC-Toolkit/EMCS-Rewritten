package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type RuinedCommand struct{}

func (cmd RuinedCommand) Name() string { return "ruined" }
func (cmd RuinedCommand) Description() string {
	return "Sends a list of all ruined towns with their time of ruin."
}

func (cmd RuinedCommand) Options() []AppCommandOpt {
	return nil
}

func (cmd RuinedCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	ruined, err := GetRuinedTowns()
	if err != nil {
		log.Printf("failed to get towns from db:\n%v", err)
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving towns from the database. Check the console.",
		})

		return err
	}

	totalCount := len(ruined)

	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, totalCount, perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(totalCount)
		items := ruined[start:end]

		descBuilder := strings.Builder{} // More performant than concat via regular Sprintf
		for idx, t := range items {
			ruinedTs := *t.Timestamps.RuinedAt
			ruinedTime := time.UnixMilli(int64(*t.Timestamps.RuinedAt))
			after72h := ruinedTime.Add(72 * time.Hour)

			// TODO: Hard coding this isn't great. Switch to a more robust automatic version of
			// new day calculation using server info like we do in the `/newday when` cmd.
			nextNewDay := time.Date(after72h.Year(), after72h.Month(), after72h.Day(), 10, 0, 0, 0, time.UTC)
			if !nextNewDay.After(after72h) {
				nextNewDay = nextNewDay.Add(24 * time.Hour)
			}

			nationName := "No Nation"
			if t.Nation.Name != nil {
				nationName = *t.Nation.Name
			}

			X, Y, Z := t.SpawnLocation()
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=6)", X, Y, Z, X, Z)

			residents := logutil.HumanizedSprintf("`%d`", t.NumResidents())
			balance := logutil.HumanizedSprintf("`%0.0f` %s", t.Bal(), shared.EMOJIS.GOLD_INGOT)
			chunks := logutil.HumanizedSprintf("`%d` %s", t.Size(), shared.EMOJIS.CHUNK)

			fmt.Fprintf(&descBuilder, "%d. **%s** (%s) fell into ruin <t:%d:R> at %s.\n"+
				"Deletion on `%s` (<t:%d:R>).\n"+
				"Mayor: `%s` Residents: %s Balance: %s Chunks: %s\n\n",
				start+idx+1, t.Name, nationName, ruinedTs/1000, locationLink,
				utils.FormatTime(nextNewDay), nextNewDay.Unix(),
				t.Mayor.Name, residents, balance, chunks,
			)
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		title := fmt.Sprintf("[%d] List of Ruined Towns | %s", totalCount, pageStr)
		desc := descBuilder.String()

		embed := discordutil.NewEmbedBuilder(&discordutil.DARK_GOLD, &title, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}

func GetRuinedTowns() ([]oapi.TownInfo, error) {
	townsStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return nil, err
	}

	towns := townsStore.Entries()
	ruined := lo.FilterMapToSlice(towns, func(_ string, t oapi.TownInfo) (oapi.TownInfo, bool) {
		return t, t.Status.Ruined
	})

	sort.Slice(ruined, func(i, j int) bool {
		return *ruined[i].Timestamps.RuinedAt < *ruined[j].Timestamps.RuinedAt
	})

	return ruined, nil
}
