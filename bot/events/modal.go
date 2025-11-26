package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/utils/discordutil"
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateModalSubmit recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionModalSubmit {
		return
	}

	customID := i.ModalSubmitData().CustomID
	parts := strings.SplitN(customID, "_", 2) // split into parts with underscore seperator
	if len(parts) == 0 {
		return
	}

	cmdName := parts[0] // grab first part, usually the name of the command
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if modalCmd, ok := cmd.(slashcommands.ModalHandler); ok {
			_ = modalCmd.HandleModal(s, i.Interaction, customID)
		}
	}
}
