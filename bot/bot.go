package bot

import (
	"emcsrw/bot/events"
	"emcsrw/database"
	"emcsrw/shared"
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

// Initializes a Database for known map m by creating it at ~cwd/db/<mapName>, then assigning all
// Stores (persisent caches) that should exist on it by creating their files inside said dir.
func InitDB(m shared.EMCMap) *database.Database {
	mdb, err := database.New("./db", m)
	if err != nil {
		log.Fatalf("Cannot initialize database for map '%s':\n%v", m, err)
	}
	log.Printf("Initialized database for map '%s'.\n", m)

	// Define all stores we want to exist on this database.
	// If a store does not exist, it is created under the ./db/<mapName> dir.
	database.AssignStore(mdb, database.SERVER_STORE)
	database.AssignStore(mdb, database.TOWNS_STORE)
	database.AssignStore(mdb, database.NATIONS_STORE)
	database.AssignStore(mdb, database.ENTITIES_STORE) // Store keys: residentlist, townlesslist
	database.AssignStore(mdb, database.PLAYERS_STORE)
	database.AssignStore(mdb, database.ALLIANCES_STORE)
	database.AssignStore(mdb, database.NEWS_STORE)
	database.AssignStore(mdb, database.USAGE_USERS_STORE)
	//database.AssignStoreToDB[map[string]any](mdb, database.USAGE_LEADERBOARD_STORE)

	return mdb
}
