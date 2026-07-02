package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

const DAY_SECS = 60 * 60 * 24

type NewDayCommand struct{}

func (cmd NewDayCommand) Name() string { return "newday" }
func (cmd NewDayCommand) Description() string {
	return "Base command for Towny new day related subcommands."
}

func (cmd NewDayCommand) Options() []AppCommandOpt {
	return []AppCommandOpt{
		discordutil.SubcommandOption("when", "Sends the amount of time until the elusive new day occurs."),
		//discordutil.SubcommandOption("falling", ""),
		//discordutil.SubcommandOption("ruined", ""),
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

	ts, secUntilNewDay := secUntilNewDay(info)

	title := "New Day | Time Information"
	desc := fmt.Sprintf(
		"The next Towny new day occurs in <t:%d:R>.\nExactly %s from now.",
		ts, utils.FormatElapsed(time.Duration(secUntilNewDay)*time.Second),
	)

	embed := discordutil.NewEmbedBuilder(&discordutil.DARK_PURPLE, &title, &desc, nil)
	discordutil.SendOrEditReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed.Build()},
	})

	return nil
}

func secUntilNewDay(info *oapi.ServerInfo) (unix int64, sec int64) {
	newDayTime := info.Timestamps.NewDayTime
	serverTod := info.Timestamps.ServerTimeOfDay

	secUntilNewDay := (newDayTime - serverTod + DAY_SECS) % DAY_SECS

	now := time.Now().Unix()
	return now + secUntilNewDay, secUntilNewDay
}

// Minecraft ticks until next in-game day
// ticksUntilMCNewDay := (newDayTime - timePassed + newDayTime) % newDayTime
