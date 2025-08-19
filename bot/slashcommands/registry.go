package slashcommands

import "github.com/bwmarrin/discordgo"

type SlashCommand interface {
	Name() string
	Description() string
	Type() discordgo.ApplicationCommandType
	Options() []*discordgo.ApplicationCommandOption
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

var commands = map[string]SlashCommand{}

func All() map[string]SlashCommand {
	return commands
}

func Register(cmd SlashCommand) {
	commands[cmd.Name()] = cmd
}

// Called before the bot runs, before the main() func.
func init() {
	Register(ServerInfoCommand{})
	Register(TownCommand{})
}
