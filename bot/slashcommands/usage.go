package slashcommands

import (
	"cmp"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type UsageCommand struct{}

func (cmd UsageCommand) Name() string { return "usage" }
func (cmd UsageCommand) Description() string {
	return "Get info on your personal bot usage or view the global usage leaderboard."
}

func (cmd UsageCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Name:        "self",
			Description: "Output info about your own bot usage statistics.",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
		{
			Name:        "leaderboard",
			Description: "View the bot usage statistics globally via a leaderboard.",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
	}
}

func (cmd UsageCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("self"); opt != nil {
		return executeSelf(s, i.Interaction)
	}
	if opt := cdata.GetOption("leaderboard"); opt != nil {
		return executeLeaderboard(s, i.Interaction)
	}

	_, err := discordutil.SendOrEditReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Content: "Error occurred getting sub command option. Somehow you sent none of them?",
	})

	return err
}

func executeSelf(s *discordgo.Session, i *discordgo.Interaction) error {
	usageStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.USAGE_USERS_STORE)
	if err != nil {
		return err
	}

	author := discordutil.GetInteractionAuthor(i)
	usage, _ := usageStore.Get(author.ID)
	if usage == nil {
		return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No usage recorded.",
		})
	}

	// Get stats for each time window and convert to formatted string.
	mostUsedStr, totalAllTime := formatCommandStats(usage, nil, 20)

	since3m := time.Now().AddDate(0, -3, 0)
	mostUsed3mStr, total3Months := formatCommandStats(usage, &since3m, 20)

	since30d := time.Now().AddDate(0, 0, -30)
	mostUsed30dStr, total30Days := formatCommandStats(usage, &since30d, 20)

	embed := &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Bot Usage Statistics | `%s`", author.Username),
		Color:  discordutil.WHITE,
		Footer: embeds.DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			discordutil.NewEmbedField("Top Commands (All Time)", fmt.Sprintf("%s\n\n%s", totalAllTime, mostUsedStr), true),
			discordutil.NewEmbedField("Top Commands (Last 3 Months)", fmt.Sprintf("%s\n\n%s", total3Months, mostUsed3mStr), true),
			discordutil.NewEmbedField("Top Commands (Last 30 Days)", fmt.Sprintf("%s\n\n%s", total30Days, mostUsed30dStr), true),
		},
	}

	return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func executeLeaderboard(s *discordgo.Session, i *discordgo.Interaction) error {
	discordutil.ReplyWithError(s, i, errors.New("Command not implemented yet."))
	return nil
}

func formatCommandStats(usage *database.UserUsage, since *time.Time, limit int) (string, string) {
	stats := []database.UsageCommandStat{}
	if since == nil {
		stats = usage.GetCommandStats()
	} else {
		stats = usage.GetCommandStatsSince(*since)
	}

	slices.SortFunc(stats, func(a, b database.UsageCommandStat) int {
		return cmp.Compare(b.Count, a.Count)
	})

	limit = min(limit, len(stats))
	mostUsed := make([]string, 0, limit)
	for _, stat := range stats[:limit] {
		cmdUsageStr := utils.HumanizedSprintf("/%s - `%d` times", stat.Name, stat.Count)
		mostUsed = append(mostUsed, cmdUsageStr)
	}

	list := strings.Join(mostUsed, "\n")
	total := utils.HumanizedSprintf("Total: `%d`", usage.CalculateTotal(stats))
	return list, total
}
