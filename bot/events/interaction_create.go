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
		if err := recover(); err != nil {
			log.Printf("handler InteractionCreateApplicationCommand recovered from a panic.\n%v\n%s", err, debug.Stack())

			errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
			content := "Bot attempted to panic during this command. Please report the following error.\n" + errStr

			// Not already deferred, reply.
			err := discordutil.Reply(s, i.Interaction, &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: content,
			})

			if err != nil {
				// Must be deferred, send follow up.
				discordutil.FollowUpContentEphemeral(s, i.Interaction, content)
			}
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
