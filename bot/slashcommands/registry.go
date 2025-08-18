package slashcommands

import "github.com/bwmarrin/discordgo"

type SlashCommand interface {
	Name() string
	Description() string
	Type() discordgo.ApplicationCommandType
	Options() []*discordgo.ApplicationCommandOption
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var commands = map[string]SlashCommand{}

func Register(cmd SlashCommand) {
	commands[cmd.Name()] = cmd
}

func All() map[string]SlashCommand {
	return commands
}
