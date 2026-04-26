package slashcommands

import (
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/logutil"
	"fmt"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type TownlessCommand struct{}

func (cmd TownlessCommand) Name() string { return "townless" }
func (cmd TownlessCommand) Description() string {
	return "Retrieve information on townless players."
}

func (cmd TownlessCommand) Options() AppCommandOpts {
	return nil
}

func (cmd TownlessCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	entitiesStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ENTITIES_STORE)
	if err != nil {
		return err
	}

	// map of uuid -> name
	townlessEntities, err := entitiesStore.Get("townlesslist")
	if err != nil {
		return err
	}

	townlessCount := len(*townlessEntities)
	townlessNames := lo.Values(*townlessEntities)
	slices.Sort(townlessNames)

	perField := 20
	perPage := perField * 4

	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, townlessCount, perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(townlessCount)

		entries := townlessNames[start:end]
		entriesCount := len(entries)

		col1End := min(perField, entriesCount)
		col2End := min(2*perField, entriesCount)
		col3End := min(3*perField, entriesCount)
		col4End := min(4*perField, entriesCount)

		// 1-40
		col1 := entries[0:col1End]       // top-left
		col3 := entries[col1End:col2End] // bottom-left

		// 41-80
		col2 := entries[col2End:col3End] // top-right
		col4 := entries[col3End:col4End] // bottom-right

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Color: discordutil.PURPLE,
			Title: logutil.HumanizedSprintf("[%d] List of Townless Players | %s", townlessCount, pageStr),
			Fields: []*discordgo.MessageEmbedField{
				// first row
				NewEmbedField("", formatColumn(col1, 0), true),
				NewEmbedFieldSpacer(true),
				NewEmbedField("", formatColumn(col2, perField*2), true),
				// second row
				NewEmbedField("", formatColumn(col3, perField), true),
				NewEmbedFieldSpacer(true),
				NewEmbedField("", formatColumn(col4, perField*3), true),
			},
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
}

func formatColumn(entries []string, offset int) string {
	lines := make([]string, len(entries))
	for i, name := range entries {
		lines[i] = fmt.Sprintf("%d. `%s`", offset+i+1, name)
	}

	return strings.Join(lines, "\n")
}
