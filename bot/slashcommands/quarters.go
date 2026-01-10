package slashcommands

import (
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils/discordutil"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

var NewEmbedField = discordutil.NewEmbedField

type QuartersCommand struct{}

func (cmd QuartersCommand) Name() string { return "quarters" }
func (cmd QuartersCommand) Description() string {
	return "For all things relating to the quarters plugin."
}

func (cmd QuartersCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "forsale",
			Description: "Retrieve a list of all quarters for sale in a town.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("name", "The name of the town to query.", 2, 40),
			},
		},
	}
}

func (cmd QuartersCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	discordutil.DeferReply(s, i.Interaction)

	data := i.ApplicationCommandData()

	subCmd := data.GetOption("forsale")
	if subCmd == nil {
		return fmt.Errorf("forsale sub cmd not found for /quarters cmd. wtf?")
	}

	townOpt := subCmd.GetOption("name")
	if townOpt == nil {
		return fmt.Errorf("no name input for /quarters forsale sub cmd. wtf?")
	}

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	town, err := townStore.Find(func(t oapi.TownInfo) bool {
		return strings.EqualFold(t.Name, townOpt.StringValue())
	})
	if err != nil {
		discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Failed to get quarters. Town `%s` does not exist.", townOpt.StringValue()),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	townQuarters, _, _ := oapi.QueryConcurrentEntities(oapi.QueryQuarters, town.Quarters)
	qfs := lo.Filter(townQuarters, func(q oapi.Quarter, _ int) bool {
		return q.Status.IsForSale
	})

	// Highest -> Lowest
	slices.SortFunc(qfs, func(a, b oapi.Quarter) int {
		return cmp.Compare(*a.Stats.Price, *b.Stats.Price)
	})

	count := len(qfs)
	if count < 1 {
		_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("No quarters for sale in town: `%s`", town.Name),
		})

		return err
	}

	perPage := 1
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, count, perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		items := qfs[start:end]
		q := items[0]

		affiliation := *q.Town.Name
		if q.Nation.Name != nil {
			affiliation += fmt.Sprintf(" (%s)", *q.Nation.Name)
		}

		owner := "No Owner"
		if q.Owner.Name != nil {
			owner = *q.Owner.Name
		}

		price := float32(0)
		if q.Stats.Price != nil {
			price = *q.Stats.Price
		}

		registeredTs := q.Timestamps.Registered / 1000 // Seconds
		registeredStr := fmt.Sprintf("<t:%d:R>", registeredTs)

		// creator := "No Creator?"
		// if q.Creator != nil {
		// 	creator = *q.Creator
		// }

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:  fmt.Sprintf("Quarters For Sale | `%s` | %s", affiliation, pageStr),
			Footer: embeds.DEFAULT_FOOTER,
			Color:  discordutil.BLURPLE,
			Fields: []*discordgo.MessageEmbedField{
				NewEmbedField("Name", fmt.Sprintf("`%s`", q.Name), true),
				NewEmbedField("Current Owner", fmt.Sprintf("`%s`", owner), true),
				NewEmbedField("Created", registeredStr, true),
				//NewEmbedField("Creator", fmt.Sprintf("`%s`", creator), true),
				NewEmbedField("Type", fmt.Sprintf("`%s`", q.Type), true),
				NewEmbedField("Embassy", fmt.Sprintf("`%t`", q.Status.IsEmbassy), true),
				NewEmbedField("Price", fmt.Sprintf("`%.0f`G %s", price, shared.EMOJIS.GOLD_INGOT), true),
			},
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
}
