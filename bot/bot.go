package bot

import (
	"emcsrw/bot/events"
	"log"

	"github.com/bwmarrin/discordgo"
)

// var DM_INTENTS = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

// !! WARNING !!
// If you do not have some of these intents, the bot may not start correctly.
// Remove any you do not require and restart the bot again.
var ALL_INTENTS = discordgo.IntentMessageContent | GUILD_INTENTS
var GUILD_INTENTS = discordgo.IntentGuilds |
	discordgo.IntentGuildMessages |
	discordgo.IntentGuildMessageReactions

// Specifys the events we want to handle.
// Make sure we actually listen and respond to them once specified to avoid errors!
//
// https://discord.com/developers/docs/events/gateway-events#receive-events
var EVENT_HANDLERS = [...]any{
	events.OnReady,
	events.OnApplicationCommandInteractionCreate,
	events.OnModalSubmitInteractionCreate,
	events.OnSelectMenuInteractionCreate,
	events.OnButtonInteractionCreate,
	events.OnAutocompleteInteractionCreate,
}

// Uses session s to open a WebSocket connection to the Discord gateway.
func Connect(s *discordgo.Session) *discordgo.Session {
	s.Identify.Intents = ALL_INTENTS
	s.SyncEvents = false // Run handlers in a goroutine to prevent a command waiting on another user's command.

	for _, eh := range EVENT_HANDLERS {
		s.AddHandler(eh)
	}

	// Open websocket connection to Discord gateway.
	err := s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	log.Println("Established connection to Discord.")
	return s
}
