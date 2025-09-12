package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// 0 for Guild, 1 for User
var integrationTypes = []discordgo.ApplicationIntegrationType{
	discordgo.ApplicationIntegrationUserInstall,
	discordgo.ApplicationIntegrationGuildInstall,
}

// 0 for Guilds, 2 for DMs, 3 for Private Channels
var contexts = []discordgo.InteractionContextType{
	discordgo.InteractionContextBotDM,
	discordgo.InteractionContextGuild,
}

var commands = map[string]SlashCommand{}

type AppCommandOpts = []*discordgo.ApplicationCommandOption
type SlashCommand interface {
	Name() string
	Description() string
	Options() AppCommandOpts
	// IntegrationTypes() *[]discordgo.ApplicationIntegrationType
	// Contexts() *[]discordgo.InteractionContextType
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

func ToApplicationCommand(cmd SlashCommand) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:             cmd.Name(),
		Description:      cmd.Description(),
		Options:          cmd.Options(),
		IntegrationTypes: &integrationTypes, // TODO: Require these to implemented by SlashCommand instead?
		Contexts:         &contexts,
		Type:             discordgo.ChatApplicationCommand,
	}
}

func All() map[string]SlashCommand {
	return commands
}

func Register(cmd SlashCommand) {
	if _, exists := commands[cmd.Name()]; exists {
		fmt.Printf("Command '%s' is already registered!\n", cmd.Name())
		return
	}

	commands[cmd.Name()] = cmd
}

// Called before the bot runs (just before main).
func init() {
	Register(ServerInfoCommand{})
	Register(TownCommand{})
	Register(NationCommand{})
	Register(PlayerCommand{})
	//Register(TownlessCommand{})
	Register(MysteryMasterCommand{})
	Register(AllianceCommand{})
	Register(VotePartyCommand{})
}

// ======================================= COMMAND TEMPLATE =======================================
// type ExampleCommand struct{}

// func (cmd ExampleCommand) Name() string { return "example" }
// func (cmd ExampleCommand) Description() string {
// 	return "This is an example description for a slash command."
// }

// func (cmd ExampleCommand) Options() []*discordgo.ApplicationCommandOption {
// 	return nil
// }

// func (cmd ExampleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
// 	return nil
// }
