package slashcommands

import (
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var SSEEventNames = map[string]string{
	"event-new-day": "New Day",

	"event-town-created":       "Town Created",
	"event-town-deleted":       "Town Deleted",
	"event-town-renamed":       "Town Renamed",
	"event-town-mayor-changed": "Town Mayor Changed",
	"event-town-merged":        "Town Merged",
	"event-town-ruined":        "Town Ruined",
	"event-town-reclaimed":     "Town Reclaimed",

	"event-nation-created":      "Nation Created",
	"event-nation-deleted":      "Nation Deleted",
	"event-nation-renamed":      "Nation Renamed",
	"event-nation-king-changed": "Nation King Changed",
	"event-nation-merged":       "Nation Merged",
}

type SSECommand struct{}

func (cmd SSECommand) Name() string { return "sse" }
func (cmd SSECommand) Description() string {
	return "Base command for subcommands relating to Server Events/SSE."
}

func (cmd SSECommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "setup",
			Description: "Sets up the current channel to receive certain chosen server events.",
		},
	}
}

func (cmd SSECommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// if err := discordutil.DeferReply(s, i.Interaction); err != nil {
	// 	return err
	// }

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("setup"); opt != nil {
		return sendSetupMultiSelect(s, i.Interaction)
	}

	return nil
}

func (cmd SSECommand) HandleSelectMenu(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	if customID == "sse_setup" {
		cdata := i.MessageComponentData()
		selected := toEventNames(cdata.Values) // convert the choices from values to their names

		// err := executeSetupChannel(selected)
		// if err != nil {
		// 	return err
		// }

		choicesStr := fmt.Sprintf("```%s```", strings.Join(selected, ", "))
		_, err := discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
			Content: "SSE setup complete. This channel will receive the following events:" + choicesStr,
		})

		return err
	}

	return nil
}

func sendSetupMultiSelect(s *discordgo.Session, i *discordgo.Interaction) error {
	menu := discordgo.SelectMenu{
		// Select menu, as other components, must have a customID, so we set it to this value.
		CustomID:    "sse_setup",
		Placeholder: "Choose the events this channel should receive 👇",
		Options: []discordgo.SelectMenuOption{
			discordutil.SelectMenuOption("New Day", "event-new-day", "When a new day occurs", true),
			// Town events
			discordutil.SelectMenuOption("Town Created", "event-town-created", "When a town is created.", true),
			discordutil.SelectMenuOption("Town Deleted", "event-town-deleted", "When a town is deleted.", true),
			discordutil.SelectMenuOption("Town Renamed", "event-town-renamed", "When a town changes its name.", false),
			discordutil.SelectMenuOption("Town Mayor Changed", "event-town-mayor-changed",
				"When a town changes mayor.", false,
			),
			discordutil.SelectMenuOption("Town Merged", "event-town-merged",
				"When a town merges with another town.", false,
			),
			discordutil.SelectMenuOption("Town Ruined", "event-town-ruined",
				"When a town falls into ruin.", true,
			),
			discordutil.SelectMenuOption("Town Reclaimed", "event-town-reclaimed",
				"When a town is reclaimed.", false,
			),
			// Nation events
			discordutil.SelectMenuOption("Nation Created", "event-nation-created", "When a nation is created.", true),
			discordutil.SelectMenuOption("Nation Deleted", "event-nation-deleted", "When a nation is deleted.", true),
			discordutil.SelectMenuOption("Nation Renamed", "event-nation-renamed", "When a nation changes its name.", false),
			discordutil.SelectMenuOption("Nation King Changed", "event-nation-king-changed",
				"When a nation changes leader aka 'king'.", false,
			),
			discordutil.SelectMenuOption("Nation Merged", "event-nation-merged",
				"When a nation merges with another nation.", false,
			),
		},
	}

	return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
		Content: "SSE Events | Channel Setup",
		Flags:   discordgo.MessageFlagsEphemeral,
		Components: []discordgo.MessageComponent{
			discordutil.SelectMenuActionRow(menu),
		},
	})
}

type ChannelID = string // Refers to a Discord channel.
type ChannelEventMap = map[ChannelID][]string

func executeSetupChannel(s *discordgo.Session, i *discordgo.Interaction, selected []string) error {
	return nil
}

func toEventNames(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = prettyEvent(v)
	}

	return out
}

func prettyEvent(v string) string {
	parts := strings.Split(strings.TrimPrefix(v, "event-"), "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}

	return strings.Join(parts, " ")
}
