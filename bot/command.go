package bot

import (
	"emcsrw/bot/slashcommands"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func GetInteractionUsername(i *discordgo.InteractionCreate) string {
	if i.User == nil {
		return i.Member.User.Username
	}

	return i.User.Username
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Regular slash command interaction
	if i.Type == discordgo.InteractionApplicationCommand {
		author := GetInteractionUsername(i)

		cmdName := i.ApplicationCommandData().Name
		cmd := slashcommands.All()[cmdName]

		err := cmd.Execute(s, i)
		if err != nil {
			fmt.Printf("'%s' failed to execute command /%s:\n%v", author, cmdName, err)
		} else {
			fmt.Printf("'%s' successfully executed command /%s", author, cmdName)
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
