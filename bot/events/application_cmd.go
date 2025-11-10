package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateApplicationCommand recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.GetInteractionAuthor(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmdType := i.ApplicationCommandData().CommandType
	cmd := slashcommands.All()[cmdName]

	start := time.Now()
	err := cmd.Execute(s, i)
	elapsed := time.Since(start)

	success := err == nil
	fmt.Println()
	if success {
		log.Printf("'%s' successfully executed command /%s (took: %s)\n", author.Username, cmdName, elapsed)
	} else {
		log.Printf("'%s' failed to execute command /%s:\n%v\n\n", author.Username, cmdName, err)
		// TODO: Maybe send error message here like we do with panic
	}

	if cmdName != "usage" {
		mdb, err := database.Get(shared.ACTIVE_MAP)
		if err != nil {
			fmt.Println()
			log.Printf("error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}

		e := database.UsageCommandEntry{
			Type:      uint8(cmdType),
			Timestamp: time.Now().Unix(),
			Success:   success,
		}

		if err := database.UpdateUsageForUser(mdb, author, cmdName, e); err != nil {
			fmt.Println()
			log.Printf("error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}
	}
}
