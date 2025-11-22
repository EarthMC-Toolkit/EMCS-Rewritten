package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type OnlineCommand struct{}

func (cmd OnlineCommand) Name() string { return "online" }
func (cmd OnlineCommand) Description() string {
	return "Base command for subcommands relating to online players."
}

func (cmd OnlineCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "town",
			Description: "Query information about the online status of a town's residents.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("name", "The name of the town to query.", 2, 40),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "nation",
			Description: "Query information about the online status of a nation's residents.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("name", "The name of the nation to query.", 2, 40),
			},
		},
	}
}

func (cmd OnlineCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()

	opt := cdata.GetOption("town")
	if opt != nil {
		return executeOnlineTown(s, i.Interaction, opt.GetOption("name").StringValue())
	}

	opt = cdata.GetOption("nation")
	if opt != nil {
		return executeOnlineNation(s, i.Interaction, opt.GetOption("name").StringValue())
	}

	_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Content: "Error occurred getting sub command option. Somehow you sent none of them?",
	})

	return err
}

func executeOnlineTown(s *discordgo.Session, i *discordgo.Interaction, townName string) error {
	discordutil.DeferReply(s, i)

	townStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	town, err := townStore.FindFirst(func(t oapi.TownInfo) bool {
		return strings.EqualFold(t.Name, townName)
	})
	if err != nil {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Failed to get online players. Town `%s` does not exist.", townName),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	online, _ := getOnlineResidents(town.Residents...)
	if len(online) < 1 {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("No players online in town: `%s`.", townName),
		})

		return err
	}

	return sendPaginator(s, i, online, 15, func(p oapi.PlayerInfo) string {
		balStr := fmt.Sprintf("%s `%0.f`G", shared.EMOJIS.GOLD_INGOT, p.Stats.Balance)
		return fmt.Sprintf("`%s` (%s) %s\n", p.Name, p.GetRank(), balStr)
	})
}

func executeOnlineNation(s *discordgo.Session, i *discordgo.Interaction, nationName string) error {
	discordutil.DeferReply(s, i)

	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	nation, err := nationStore.FindFirst(func(n oapi.NationInfo) bool {
		return strings.EqualFold(n.Name, nationName)
	})
	if err != nil {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Failed to get online players. Nation `%s` does not exist.", nationName),
		})

		return err
	}

	online, _ := getOnlineResidents(nation.Residents...)
	if len(online) < 1 {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("No players online in nation: `%s`.", nationName),
		})

		return err
	}

	return sendPaginator(s, i, online, 15, func(p oapi.PlayerInfo) string {
		balStr := fmt.Sprintf("%s `%0.f`G", shared.EMOJIS.GOLD_INGOT, p.Stats.Balance)
		return fmt.Sprintf("`%s` of **%s** (%s) %s\n", p.Name, *p.Town.Name, p.GetRank(), balStr)
	})
}

func getOnlineResidents(entities ...oapi.Entity) ([]oapi.PlayerInfo, error) {
	residents, errs, _ := oapi.QueryConcurrentEntities(oapi.QueryPlayers, entities)
	online := lo.Filter(residents, func(p oapi.PlayerInfo, _ int) bool {
		return p.Status.IsOnline
	})

	return online, errors.Join(errs...)
}

func sendPaginator(
	s *discordgo.Session, i *discordgo.Interaction,
	players []oapi.PlayerInfo, perPage int,
	contentFunc func(p oapi.PlayerInfo) string,
) error {
	count := len(players)

	// Alphabet sort by player name
	slices.SortStableFunc(players, func(a oapi.PlayerInfo, b oapi.PlayerInfo) int {
		if a.Name > b.Name {
			return 1
		}

		return -1
	})

	paginator := discordutil.NewInteractionPaginator(s, i, count, perPage).
		WithTimeout(5 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(count)

		content := ""
		for _, p := range players[start:end] {
			content += contentFunc(p)
		}

		data.Content = content
		if paginator.TotalPages() > 1 {
			data.Content += fmt.Sprintf("\nPage %d/%d", curPage+1, paginator.TotalPages())
		}
	}

	return paginator.Start()
}
