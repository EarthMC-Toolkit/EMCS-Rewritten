package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

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

	townStore, err := store.GetStoreForMap[oapi.TownInfo](common.ACTIVE_MAP, "towns")
	if err != nil {
		return err
	}

	town, err := townStore.Find(func(t oapi.TownInfo) bool {
		return strings.EqualFold(t.Name, townOpt.StringValue())
	})
	if err != nil {
		return err
	}

	townQuarters, _, _ := oapi.QueryConcurrentEntities(oapi.QueryQuarters, town.Quarters)
	qfs := lo.Filter(townQuarters, func(q oapi.Quarter, _ int) bool {
		return q.Status.IsForSale
	})

	count := len(qfs)
	if count < 1 {
		_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("No quarters for sale in town: `%s`", town.Name),
		})

		return err
	}

	perPage := 1

	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, count, perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		items := qfs[start:end]
		q := items[0]

		registeredTs := q.Timestamps.Registered / 1000 // Seconds
		registeredStr := fmt.Sprintf("<t:%d:R>", registeredTs)

		var owner = "No Owner"
		if q.Owner.Name != nil {
			owner = *q.Owner.Name
		}

		var price float32
		if q.Stats.Price != nil {
			price = *q.Stats.Price
		}

		var creator = "No Creator?"
		if q.Creator != nil {
			creator = *q.Creator
		}

		pageStr := fmt.Sprintf("Quarter %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:  pageStr + fmt.Sprintf(" | `%s`", q.Name),
			Footer: common.DEFAULT_FOOTER,
			Color:  discordutil.BLURPLE,
			Fields: []*discordgo.MessageEmbedField{
				common.NewEmbedField("Registered", registeredStr, true),
				common.NewEmbedField("Creator", fmt.Sprintf("`%s`", creator), true),
				common.NewEmbedField("Owner", fmt.Sprintf("`%s`", owner), true),
				common.NewEmbedField("Type", fmt.Sprintf("`%s`", q.Type), true),
				common.NewEmbedField("Is Embassy?", fmt.Sprintf("`%t`", q.Status.IsEmbassy), true),
				common.NewEmbedField("Price", fmt.Sprintf("`%.0f`G %s", price, common.EMOJIS.GOLD_INGOT), true),
			},
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
		data.Components = []discordgo.MessageComponent{
			paginator.NewNavigationButtonRow(),
		}
	}

	return paginator.Start()
}
