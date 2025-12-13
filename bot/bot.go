package bot

import (
	"emcsrw/bot/events"
	"emcsrw/database"
	"emcsrw/shared"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

//var dmIntents = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

var GUILD_INTENTS = discordgo.IntentGuilds |
	discordgo.IntentGuildMessages |
	discordgo.IntentGuildMessageReactions

var ALL_INTENTS = discordgo.IntentMessageContent | GUILD_INTENTS

func Run(botToken string) (*discordgo.Session, *database.Database) {
	// Initialize a Discord Session
	s, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}

	// Never run handlers synchronously, always run them in a goroutine.
	s.SyncEvents = false

	// Register funcs that handle specific gateway events.
	// https://discord.com/developers/docs/events/gateway-events#receive-events
	s.AddHandler(events.OnReady)
	s.AddHandler(events.OnInteractionCreateApplicationCommand) // Slash cmds
	s.AddHandler(events.OnInteractionCreateModalSubmit)
	s.AddHandler(events.OnInteractionCreateButton)
	s.AddHandler(events.OnInteractionCreateAutocomplete)
	//s.AddHandler(events.OnInteractionCreateSelectMenu)

	s.Identify.Intents = ALL_INTENTS

	fmt.Printf("\nInitializing map databases..\n")

	auroraDB, err := database.New("./db", shared.SUPPORTED_MAPS.AURORA)
	if err != nil {
		log.Fatalf("Cannot initialize database for map '%s':\n%v", shared.SUPPORTED_MAPS.AURORA, err)
	}

	// Define all stores we want to exist on Aurora database.
	// If a store does not exist, it is created under the ./db/aurora dir.
	database.AssignStore(auroraDB, database.SERVER_STORE)
	database.AssignStore(auroraDB, database.TOWNS_STORE)
	database.AssignStore(auroraDB, database.NATIONS_STORE)
	database.AssignStore(auroraDB, database.ENTITIES_STORE) // Store keys: residentlist, townlesslist
	database.AssignStore(auroraDB, database.PLAYERS_STORE)
	database.AssignStore(auroraDB, database.ALLIANCES_STORE)
	database.AssignStore(auroraDB, database.USAGE_USERS_STORE)
	//database.AssignStoreToDB[map[string]any](auroraDB, database.USAGE_LEADERBOARD_STORE)

	fmt.Printf("\n\nEstablishing connection to Discord..\n")

	// Open WS connection to Discord.
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	return s, auroraDB
}
