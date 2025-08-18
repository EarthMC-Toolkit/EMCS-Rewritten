package bot

import (
	"emcsrw/bot/slashcommands"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Regular slash command interaction
	if i.Type == discordgo.InteractionApplicationCommand {
		cmdName := i.ApplicationCommandData().Name
		cmd := slashcommands.All()[cmdName]

		err := cmd.Execute(s, i)
		if err != nil {
			fmt.Printf("Failed to execute command '%s':\n%v", cmdName, err)
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
