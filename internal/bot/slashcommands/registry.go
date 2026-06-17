package slashcommands

import (
	"emcsrw/pkg/utils/logutil"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var commands = make(map[string]SlashCommand)

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

type SlashCommand interface {
	Name() string
	Description() string
	Options() []AppCommandOpt
	// IntegrationTypes() *[]discordgo.ApplicationIntegrationType
	// Contexts() *[]discordgo.InteractionContextType
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

type SelectMenuHandler interface {
	HandleSelectMenu(s *discordgo.Session, i *discordgo.Interaction, customID string) error
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
func SyncRemote(s *discordgo.Session, appID, guildID string) (local []*discordgo.ApplicationCommand, created []*discordgo.ApplicationCommand) {
	for _, cmd := range commands {
		local = append(local, ToApplicationCommand(cmd))
	}

	created, err := s.ApplicationCommandBulkOverwrite(appID, guildID, local)
	if err != nil {
		logutil.Printf(logutil.RED, "Failed to sync slash commands! Error occurred during bulk overwrite:\n%s\n", err)
		return
	}

	logutil.Println(logutil.GREEN, "Successfully synced slash commands.")
	logutil.Space()
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
	if len(cmd.Name()) > 32 {
		fmt.Printf("Error registering command with invalid name: '%s'. Must be 1-32 chars.", cmd.Name())
	}
	if len(cmd.Description()) > 100 {
		fmt.Printf("Error registering command '%s'. Description must be 1-100 chars.", cmd.Name())
	}

	commands[cmd.Name()] = cmd
}

// Called before the bot runs (just before main).
func init() {
	RegisterAllCommands()
}

func RegisterAllCommands() {
	// Main (Player)
	Register(PlayerCommand{})
	Register(ResidentCommand{})
	Register(OnlineCommand{})
	Register(TownlessCommand{})
	Register(VisibleCommand{})

	// Main (Other)
	Register(TownCommand{})
	Register(NationCommand{})
	Register(AllianceCommand{})
	Register(RuinedCommand{})
	Register(FallingCommand{})
	Register(QuartersCommand{})

	// Util
	Register(ServerCommand{})
	Register(NewsCommand{})
	Register(RouteCommand{})
	Register(VotePartyCommand{})
	Register(NewDayCommand{})
	Register(MysteryMasterCommand{})
	Register(SSECommand{})

	// Misc
	Register(DevCommand{})
	Register(UsageCommand{})
}

// ======================================= COMMAND TEMPLATE =======================================
// type ExampleCommand struct{}

// func (cmd ExampleCommand) Name() string { return "example" }
// func (cmd ExampleCommand) Description() string {
// 	return "This is an example description for a slash command."
// }

// func (cmd ExampleCommand) Options() []AppCommandOpt {
// 	return nil
// }

// func (cmd ExampleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleModal(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	return nil
// }

// func (cmd ExampleCommand) HandleSelectMenu(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
// 	return nil
// }
