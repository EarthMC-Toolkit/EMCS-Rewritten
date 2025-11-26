// List of message components: https://discord.com/developers/docs/components/reference#component-object-component-types
package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/utils/discordutil"
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateButton recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	if data.ComponentType != discordgo.ButtonComponent {
		return
	}

	parts := strings.SplitN(data.CustomID, "_", 2)
	if len(parts) == 0 {
		return
	}

	cmdName := parts[0]
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if buttonCmd, ok := cmd.(slashcommands.ButtonHandler); ok {
			_ = buttonCmd.HandleButton(s, i.Interaction, data.CustomID)
		}
	}
}

// func OnInteractionCreateSelectMenu(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			log.Printf("handler OnInteractionCreateSelectMenu recovered from a panic.\n%v\n%s", err, debug.Stack())
// 			discordutil.ReplyWithPanicError(s, i.Interaction, err)
// 		}
// 	}()

// 	if i.Type != discordgo.InteractionMessageComponent {
// 		return
// 	}

// 	data := i.MessageComponentData()
// 	if data.ComponentType != discordgo.SelectMenuComponent {
// 		return
// 	}

// }
