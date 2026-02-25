package slashcommands

import (
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

const secInADay = 86400

type NewDayCommand struct{}

func (cmd NewDayCommand) Name() string { return "newday" }
func (cmd NewDayCommand) Description() string {
	return "Base command for Towny new day related subcommands."
}

func (cmd NewDayCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Name:        "when",
			Description: "Sends the amount of time until the elusive new day occurs.",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
	}
}

func (cmd NewDayCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("when"); opt != nil {
		return executeNewDayWhen(s, i.Interaction)
	}

	return nil
}

func executeNewDayWhen(s *discordgo.Session, i *discordgo.Interaction) error {
	// grab new day time
	serverStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.SERVER_STORE)
	if err != nil {
		return err
	}

	info, err := serverStore.Get("info")
	if err != nil {
		log.Printf("failed to get serverinfo from db:\n%v", err)
		return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
			Content: "An error occurred retrieving server info from the database. Check the console.",
		})
	}

	newDayTime := info.Timestamps.NewDayTime
	serverTod := info.Timestamps.ServerTimeOfDay

	secUntilNewDay := (newDayTime - serverTod + secInADay) % secInADay
	now := time.Now().Unix()

	sec := now + secUntilNewDay
	embed := &discordgo.MessageEmbed{
		Title: "New Day | Time Information",
		Description: fmt.Sprintf(
			"The next Towny new day occurs in <t:%d:R>.\nExactly %s from now.",
			sec, FormatDuration(secUntilNewDay),
		),
		Footer: embeds.DEFAULT_FOOTER,
		Color:  discordutil.DARK_PURPLE,
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})

	return nil
}

func FormatDuration(secs int64) string {
	hours := secs / 3600
	minutes := (secs % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("`%dhrs`, `%dm` and `%ds`", hours, minutes, secs%60)
	}
	if minutes > 0 {
		return fmt.Sprintf("`%dm` and `%ds`", minutes, secs%60)
	}

	return fmt.Sprintf("`%ds`", secs%60)
}

// Minecraft ticks until next in-game day
// ticksUntilMCNewDay := (newDayTime - timePassed + newDayTime) % newDayTime
