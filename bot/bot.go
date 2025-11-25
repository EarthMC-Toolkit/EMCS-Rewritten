package bot

import (
	"emcsrw/bot/events"
	"emcsrw/database"
	"emcsrw/shared"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

//var dmIntents = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

var GUILD_INTENTS = discordgo.IntentGuilds |
	discordgo.IntentGuildMessages |
	discordgo.IntentGuildMessageReactions

var ALL_INTENTS = discordgo.IntentMessageContent | GUILD_INTENTS

func Run(botToken string) {
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
	s.AddHandler(events.OnInteractionCreateMessageComponent)   // Buttons, rows, select menus
	s.AddHandler(events.OnInteractionCreateModalSubmit)        // Modal submit

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
	database.AssignStore(auroraDB, database.ALLIANCES_STORE)
	database.AssignStore(auroraDB, database.USAGE_USERS_STORE)
	//database.AssignStoreToDB[map[string]any](auroraDB, database.USAGE_LEADERBOARD_STORE)

	fmt.Printf("\n\nEstablishing connection to Discord..\n")

	// Open WS connection to Discord.
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	//go capi.Serve()

	// Wait for Ctrl+C or kill.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))

	// Since the `defer` keyword only works in successful exits,
	// closing explicitly here makes sure we always properly cleanup.
	if err := auroraDB.Flush(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
	if err := s.Close(); err != nil {
		log.Printf("error closing Discord session: %v", err)
	}
}
