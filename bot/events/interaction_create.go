package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Regular slash command interaction
	if i.Type == discordgo.InteractionApplicationCommand {
		author := utils.UserFromInteraction(i.Interaction)

		cmdName := i.ApplicationCommandData().Name
		cmd := slashcommands.All()[cmdName]

		err := cmd.Execute(s, i)
		if err != nil {
			fmt.Printf("'%s' failed to execute command /%s:\n%v", author.Username, cmdName, err)
		} else {
			fmt.Printf("'%s' successfully executed command /%s", author.Username, cmdName)
		}

		return
	}

	// Autocomplete interaction
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		return
	}

	// Modal interaction
	if i.Type == discordgo.InteractionModalSubmit {
		return
	}
}
