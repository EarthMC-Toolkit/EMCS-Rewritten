package bot

import (
	"emcsrw/bot/events"
	"emcsrw/database"
	"emcsrw/shared"
	"log"

	"github.com/bwmarrin/discordgo"
)

// !! WARNING !!
// If you do not have some of these intents, the bot may not start correctly.
// Remove any you do not require and restart the bot again.
var ALL_INTENTS = discordgo.IntentMessageContent | GUILD_INTENTS
var GUILD_INTENTS = discordgo.IntentGuilds |
	discordgo.IntentGuildMessages |
	discordgo.IntentGuildMessageReactions

//var DM_INTENTS = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

// Initializes the database for the active map, instantly
// exiting and outputting to stdout in case of an error.
//
// Once complete, we open a WS connection to s after setting intents and adding event handlers.
func Run(s *discordgo.Session) (*discordgo.Session, *database.Database) {
	mdb, err := database.New("./db", shared.ACTIVE_MAP)
	if err != nil {
		log.Fatalf("Cannot initialize database for active map '%s':\n%v", shared.ACTIVE_MAP, err)
	}
	log.Printf("\nInitialized database for active map '%s'.\n", shared.ACTIVE_MAP)

	// Define all stores we want to exist on Aurora database.
	// If a store does not exist, it is created under the ./db/aurora dir.
	database.AssignStore(mdb, database.SERVER_STORE)
	database.AssignStore(mdb, database.TOWNS_STORE)
	database.AssignStore(mdb, database.NATIONS_STORE)
	database.AssignStore(mdb, database.ENTITIES_STORE) // Store keys: residentlist, townlesslist
	database.AssignStore(mdb, database.PLAYERS_STORE)
	database.AssignStore(mdb, database.ALLIANCES_STORE)
	database.AssignStore(mdb, database.USAGE_USERS_STORE)
	//database.AssignStoreToDB[map[string]any](auroraDB, database.USAGE_LEADERBOARD_STORE)

	s.Identify.Intents = ALL_INTENTS
	s.SyncEvents = false // Never run handlers synchronously, always run them in a goroutine.

	// Register funcs that handle specific gateway events.
	// https://discord.com/developers/docs/events/gateway-events#receive-events
	s.AddHandler(events.OnReady)
	s.AddHandler(events.OnInteractionCreateApplicationCommand) // Slash cmds
	s.AddHandler(events.OnInteractionCreateModalSubmit)
	s.AddHandler(events.OnInteractionCreateButton)
	s.AddHandler(events.OnInteractionCreateAutocomplete)

	// Open websocket connection to Discord gateway.
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}
	log.Printf("\n\nEstablished connection to Discord.\n")

	return s, mdb
}
