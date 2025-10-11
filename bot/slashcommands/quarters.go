package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils/discordutil"
	"fmt"

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
		return t.Name == townOpt.StringValue()
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

		pageStr := fmt.Sprintf("Quarter %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:  pageStr + fmt.Sprintf(" | `%s`", q.Name),
			Footer: common.DEFAULT_FOOTER,
			Color:  discordutil.DARK_GOLD,
			Fields: []*discordgo.MessageEmbedField{
				common.NewEmbedField("Owner", fmt.Sprintf("`%s`", *q.Owner.Name), true),
				common.NewEmbedField("Type", fmt.Sprintf("`%s`", string(q.Type)), true),
				common.NewEmbedField("Price", fmt.Sprintf("`%d`", *q.Stats.Price), true),
				common.NewEmbedField("Registered", registeredStr, false),
			},
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
		data.Components = []discordgo.MessageComponent{
			paginator.NewNavigationButtonRow(),
		}
	}

	return paginator.Start()
}
