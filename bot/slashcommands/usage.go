package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/bot/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
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
		return ExecuteSelf(s, i.Interaction)
	}
	if opt := cdata.GetOption("leaderboard"); opt != nil {
		return ExecuteLeaderboard(s, i.Interaction)
	}

	_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Content: "Error occurred getting sub command option. Somehow you sent none of them?",
	})

	return err
}

func ExecuteSelf(s *discordgo.Session, i *discordgo.Interaction) error {
	mdb, err := store.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	if err != nil {
		return err
	}

	usageStore, err := store.GetStore[store.UserUsage](mdb, "usage-users")
	if err != nil {
		return err
	}

	author := discordutil.GetInteractionAuthor(i)
	usage, _ := usageStore.GetKey(author.ID)
	// if err != nil {
	// 	log.Printf("failed to get user usage for %s (%s):\n%v", author.Username, author.ID, err)
	// 	discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
	// 		Content: "Error occurred getting usage statistics from db.",
	// 	})
	// }

	if usage == nil {
		return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No usage recorded.",
		})
	}

	// Get stats for all time and convert to formatted string.
	statsAllTime := usage.GetCommandStats()
	top := min(20, len(statsAllTime)) // How many "most used commands" to display.
	mostUsed := make([]string, 0, top)
	for _, stat := range statsAllTime[:top] {
		mostUsed = append(mostUsed, utils.HumanizedSprintf("/%s - `%d` times", stat.Name, stat.Count))
	}
	mostUsedStr := strings.Join(mostUsed, "\n")

	// Get stats for last 30d and convert to formatted string.
	statsLast30Days := usage.GetCommandStatsSince(time.Now().AddDate(0, 0, -30))
	top = min(20, len(statsLast30Days)) // How many "most used commands" to display.
	mostUsed = make([]string, 0, top)
	for _, stat := range statsLast30Days[:top] {
		mostUsed = append(mostUsed, utils.HumanizedSprintf("/%s - `%d` times", stat.Name, stat.Count))
	}
	mostUsed30DaysStr := strings.Join(mostUsed, "\n")

	embed := &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Bot Usage Statistics | `%s`", author.Username),
		Footer: common.DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			discordutil.NewEmbedField("Total Commands Executed", utils.HumanizedSprintf("`%d`", usage.TotalCommandsExecuted()), false),
			discordutil.NewEmbedField("Top Commands (All Time)", mostUsedStr, true),
			discordutil.NewEmbedField("Top Commands (Last 30 Days)", mostUsed30DaysStr, true),
		},
		Color: discordutil.WHITE,
	}

	return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func ExecuteLeaderboard(s *discordgo.Session, i *discordgo.Interaction) error {
	return nil
}
