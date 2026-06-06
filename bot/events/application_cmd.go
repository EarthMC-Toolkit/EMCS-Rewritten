package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/logutil"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
)

func OnApplicationCommandInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnApplicationCommandInteractionCreate recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.InteractionAuthor(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmdType := i.ApplicationCommandData().CommandType
	cmd := slashcommands.All()[cmdName]

	start := time.Now()
	err := cmd.Execute(s, i)
	elapsed := time.Since(start)

	success := err == nil
	fmt.Println()
	if success {
		logutil.Printf(logutil.GREEN, "'%s' successfully executed command /%s (took: %s)\n", author.Username, cmdName, elapsed)
	} else {
		logutil.Printf(logutil.YELLOW, "'%s' failed to execute command /%s:\n\t%v\n\n", author.Username, cmdName, err)
		discordutil.ReplyWithGenericError(s, i.Interaction)
	}

	// Update usage for this cmd regardless of success/failure.
	if cmdName != "usage" {
		usageStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.USAGE_USERS_STORE)
		if err != nil {
			fmt.Println()
			logutil.Printf(logutil.RED, "error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}

		e := database.UsageCommandEntry{
			Type:      uint8(cmdType),
			Timestamp: time.Now().Unix(),
			Success:   success,
		}

		if err := database.UpdateUserUsage(usageStore, author, cmdName, e); err != nil {
			fmt.Println()
			logutil.Printf(logutil.RED, "error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}
	}
}

func OnAutocompleteInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnAutocompleteInteractionCreate recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommandAutocomplete {
		return
	}

	cmdName := i.ApplicationCommandData().Name
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if autocompleteCmd, ok := cmd.(slashcommands.AutocompleteHandler); ok {
			_ = autocompleteCmd.HandleAutocomplete(s, i.Interaction)
		}
	}
}
