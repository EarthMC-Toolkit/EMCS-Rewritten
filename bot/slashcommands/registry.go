package slashcommands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Leave empty to register commands globally
const guildID = ""

var commands = map[string]SlashCommand{}

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

type AppCommandOpt = *discordgo.ApplicationCommandOption
type AppCommandOpts = []AppCommandOpt

type SlashCommand interface {
	Name() string
	Description() string
	Options() AppCommandOpts
	// IntegrationTypes() *[]discordgo.ApplicationIntegrationType
	// Contexts() *[]discordgo.InteractionContextType
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error // TODO: Refactor to discordgo.Interaction
}

type ModalHandler interface {
	HandleModal(s *discordgo.Session, i *discordgo.Interaction, customID string) error
}

type ButtonHandler interface {
	HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error
}

type AutocompleteHandler interface {
	HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error
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

// Syncs the local slash command map with the Discord remote by creating
// them if they do not exist, or overwriting them if they do.
func SyncRemote(s *discordgo.Session) (local []*discordgo.ApplicationCommand, created []*discordgo.ApplicationCommand) {
	for _, cmd := range commands {
		local = append(local, ToApplicationCommand(cmd))
	}

	created, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, local)
	if err != nil {
		fmt.Printf("Failed to sync slash commands. Error occurred during bulk overwrite: %v\n", err)
		return
	}

	fmt.Println("Successfully synced slash commands.")
	return
}

func All() map[string]SlashCommand {
	return commands
}

func AllNames() (names []string) {
	for name := range commands {
		names = append(names, name)
	}

	return
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
	Register(AllianceCommand{})
	Register(TownCommand{})
	Register(NationCommand{})
	Register(PlayerCommand{})
	Register(ResidentCommand{})
	Register(VisibleCommand{})
	Register(OnlineCommand{})
	Register(RuinedCommand{})
	//Register(FallingCommand{})
	Register(MysteryMasterCommand{})
	Register(ServerCommand{})
	Register(VotePartyCommand{})
	Register(UsageCommand{})
	Register(QuartersCommand{})
	Register(NewDayCommand{})
}

// ======================================= COMMAND TEMPLATE =======================================
// type ExampleCommand struct{}

// func (cmd ExampleCommand) Name() string { return "example" }
// func (cmd ExampleCommand) Description() string {
// 	return "This is an example description for a slash command."
// }

// func (cmd ExampleCommand) Options() AppCommandOpts {
// 	return nil
// }

// func (cmd ExampleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleModal(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	return nil
// }
