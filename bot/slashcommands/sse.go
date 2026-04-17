package slashcommands

import (
	"emcsrw/utils/discordutil"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var sseSetupState sync.Map // Discord channel id -> selected events

type SSECommand struct{}

func (cmd SSECommand) Name() string { return "sse" }
func (cmd SSECommand) Description() string {
	return "Base command for subcommands relating to Server Events/SSE."
}

func (cmd SSECommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "configure",
			Description: "Sets up the current channel to receive certain chosen server events.",
		},
	}
}

func (cmd SSECommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// if err := discordutil.DeferReply(s, i.Interaction); err != nil {
	// 	return err
	// }

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("configure"); opt != nil {
		if i.Member != nil {
			if !discordutil.HasChannelPerm(i.Member, discordgo.PermissionManageChannels) {
				return discordutil.SendReply(s, i.Interaction, &discordgo.InteractionResponseData{
					Content: "You do not have the Manage Channel permission required to configure SSE.",
					Flags:   discordgo.MessageFlagsEphemeral,
				})
			}
		}

		return sendSetupMultiSelect(s, i.Interaction)
	}

	return nil
}

// Fires every time we interact with the select menu
func (cmd SSECommand) HandleSelectMenu(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	if customID != "sse_setup" {
		return nil
	}

	selected := i.MessageComponentData().Values
	sseSetupState.Store(i.ChannelID, selected)

	menuRow := buildSelectMenu(selected)
	btnRow := discordutil.ButtonActionRow(discordgo.Button{
		CustomID: "sse_confirm",
		Label:    "Confirm",
		Style:    discordgo.SuccessButton,
	})

	return discordutil.UpdateComponent(s, i, &discordgo.InteractionResponseData{
		Components: []discordgo.MessageComponent{menuRow, btnRow},
	})
}

func (cmd SSECommand) HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	if customID != "sse_confirm" {
		return nil
	}

	events, ok := sseSetupState.Load(i.ChannelID)
	if !ok {
		return discordutil.UpdateComponent(s, i, &discordgo.InteractionResponseData{
			Content: "No selection found.",
		})
	}

	selected := toEventNames(events.([]string))
	//confirmSetup()

	choicesStr := fmt.Sprintf("```%s```", strings.Join(selected, ", "))
	return discordutil.UpdateComponent(s, i, &discordgo.InteractionResponseData{
		Content:    "SSE setup complete. This channel will receive the following events:" + choicesStr,
		Components: []discordgo.MessageComponent{},
	})
}

func sendSetupMultiSelect(s *discordgo.Session, i *discordgo.Interaction) error {
	selectedAny, _ := sseSetupState.Load(i.ChannelID)
	selected := []string{}
	if selectedAny != nil {
		selected = selectedAny.([]string)
	}

	menuRow := buildSelectMenu(selected)
	btnRow := discordutil.ButtonActionRow(discordgo.Button{
		CustomID: "sse_confirm",
		Label:    "Confirm",
		Style:    discordgo.SuccessButton,
	})

	return discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
		Content:    "Server Events (SSE) | Channel Configuration",
		Components: []discordgo.MessageComponent{menuRow, btnRow},
	})
}

func buildSelectMenu(selected []string) discordgo.ActionsRow {
	menuOpts := []discordgo.SelectMenuOption{
		menuOption("New Day", "event-new-day", "When a new day occurs", selected),
		// Town events
		menuOption("Town Created", "event-town-created", "When a town is created.", selected),
		menuOption("Town Deleted", "event-town-deleted", "When a town is deleted.", selected),
		menuOption("Town Renamed", "event-town-renamed", "When a town changes its name.", selected),
		menuOption("Town Mayor Changed", "event-town-mayor-changed", "When a town changes mayor.", selected),
		menuOption("Town Merged", "event-town-merged", "When a town merges with another town.", selected),
		menuOption("Town Ruined", "event-town-ruined", "When a town falls into ruin.", selected),
		menuOption("Town Reclaimed", "event-town-reclaimed", "When a town is reclaimed.", selected),
		// Nation events
		menuOption("Nation Created", "event-nation-created", "When a nation is created.", selected),
		menuOption("Nation Deleted", "event-nation-deleted", "When a nation is deleted.", selected),
		menuOption("Nation Renamed", "event-nation-renamed", "When a nation changes its name.", selected),
		menuOption("Nation King Changed", "event-nation-king-changed", "When a nation changes leader aka 'king'.", selected),
		menuOption("Nation Merged", "event-nation-merged", "When a nation merges with another nation.", selected),
	}

	var min, max = 1, len(menuOpts)
	return discordutil.SelectMenuActionRow(discordgo.SelectMenu{
		// Select menu, as other components, must have a customID, so we set it to this value.
		CustomID:    "sse_setup",
		Placeholder: "Choose the events this channel should receive 👇",
		MinValues:   &min,
		MaxValues:   max,
		Options:     menuOpts,
	})
}

// TODO: Implement this.
// Create a db file with mapping of channel id -> selected events, then use
// said db file to listen to the events and post to the associated channel.
func confirmSetup(s *discordgo.Session, i *discordgo.Interaction, selected []string) error {
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

func menuOption(label, value, desc string, selected []string) discordgo.SelectMenuOption {
	return discordutil.SelectMenuOption(label, value, desc, slices.Contains(selected, value))
}
