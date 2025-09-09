package events

import (
	"emcsrw/bot/discordutil"
	"emcsrw/bot/slashcommands"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handler InteractionCreateApplicationCommand recovered from a panic.\n%v\n%s", r, debug.Stack())
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.UserFromInteraction(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmd := slashcommands.All()[cmdName]

	err := cmd.Execute(s, i)
	if err != nil {
		fmt.Printf("\n'%s' failed to execute command /%s:\n%v", author.Username, cmdName, err)
	} else {
		fmt.Printf("\n'%s' successfully executed command /%s", author.Username, cmdName)
	}
}

// List of message components: https://discord.com/developers/docs/components/reference#component-object-component-types
func OnInteractionCreateMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

}
