package bot

import (
	"emcsrw/bot/slashcommands"

	"github.com/bwmarrin/discordgo"
)

var SlashCommands = map[string]slashcommands.SlashCommand{}

func RegisterSlashCommand(cmd slashcommands.SlashCommand) {
	SlashCommands[cmd.Name()] = cmd
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		cmdName := i.ApplicationCommandData().Name
		cmd := slashcommands.All()[cmdName]

		cmd.Execute(s, i)

		return
	}
}
