package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/discordutil"
	"emcsrw/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
)

type UsageCommand struct{}

func (cmd UsageCommand) Name() string { return "usage" }
func (cmd UsageCommand) Description() string {
	return "Get info on your personal bot usage or view the global usage leaderboard."
}

func (cmd UsageCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
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
	data := i.ApplicationCommandData()
	if self := data.GetOption("self"); self != nil {
		return ExecuteSelf(s, i.Interaction)
	}
	if leaderboard := data.GetOption("leaderboard"); leaderboard != nil {
		return ExecuteLeaderboard(s, i.Interaction)
	}

	return nil
}

func ExecuteSelf(s *discordgo.Session, i *discordgo.Interaction) error {
	author := discordutil.UserFromInteraction(i)

	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	usage, err := database.GetUserUsage(db, author.ID)
	if err != nil && err != badger.ErrKeyNotFound {
		fmt.Printf("failed to get user usage for %s (%s):\n%v", author.Username, author.ID, err)
		discordutil.Reply(s, i, &discordgo.InteractionResponseData{
			Content: "Error occurred getting usage statistics from db.",
		})
	}

	if len(usage.CommandHistory) < 1 {
		return discordutil.Reply(s, i, &discordgo.InteractionResponseData{
			Content: "No usage recorded.",
		})
	}

	stats := usage.GetCommandStats()

	top := min(len(stats), 5)
	mostUsed := make([]string, 0, top)

	for _, stat := range stats[:top] {
		mostUsed = append(mostUsed, utils.HumanizedSprintf("/%s - `%d` times", stat.Name, stat.Count))
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Bot Usage Statistics | `%s`", author.Username),
		Fields: []*discordgo.MessageEmbedField{
			common.EmbedField("Most Used Commands", strings.Join(mostUsed, "\n"), false),
			common.EmbedField("Total Commands Executed", utils.HumanizedSprintf("`%d`", usage.TotalCommandsExecuted()), false),
		},
		Color: discordutil.WHITE,
	}

	return discordutil.Reply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func ExecuteLeaderboard(s *discordgo.Session, i *discordgo.Interaction) error {
	return nil
}
