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

func OnButtonInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnButtonInteractionCreate recovered from a panic.\n%v\n%s", err, debug.Stack())
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

	cmdName, _, _ := strings.Cut(data.CustomID, "_")
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if buttonCmd, ok := cmd.(slashcommands.ButtonHandler); ok {
			if err := buttonCmd.HandleButton(s, i.Interaction, data.CustomID); err != nil {
				log.Println(err)
			}
		}
	}
}

func OnSelectMenuInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnSelectMenuInteractionCreate recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	if data.ComponentType != discordgo.SelectMenuComponent {
		return
	}

	cmdName, _, _ := strings.Cut(data.CustomID, "_")
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if selectMenuCmd, ok := cmd.(slashcommands.SelectMenuHandler); ok {
			if err := selectMenuCmd.HandleSelectMenu(s, i.Interaction, data.CustomID); err != nil {
				log.Println(err)
			}
		}
	}
}
