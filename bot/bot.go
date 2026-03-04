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
var EVENT_HANDLERS = []any{
	events.OnReady,
	events.OnInteractionCreateApplicationCommand,
	events.OnInteractionCreateModalSubmit,
	events.OnInteractionCreateButton,
	events.OnInteractionCreateAutocomplete,
}

// Uses session s to open a WebSocket connection to the Discord gateway.
func Connect(s *discordgo.Session) *discordgo.Session {
	s.Identify.Intents = ALL_INTENTS
	s.SyncEvents = false // Never run handlers synchronously, always run them in a goroutine.

	for _, e := range EVENT_HANDLERS {
		s.AddHandler(e)
	}

	// Open websocket connection to Discord gateway.
	err := s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	log.Println("Established connection to Discord.")
	return s
}
