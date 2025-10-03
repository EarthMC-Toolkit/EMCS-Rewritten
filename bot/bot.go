package bot

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/events"
	"emcsrw/bot/store"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	dgo "github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

//const RED, YELLOW int = 8858420, 15844367

var guildIntents = dgo.IntentGuilds | dgo.IntentGuildMessages | dgo.IntentGuildMessageReactions

//var dmIntents = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

func Run(botToken string) {
	// Initialize a Discord Session
	s, err := dgo.New("Bot " + botToken)
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

	s.Identify.Intents = dgo.IntentMessageContent | guildIntents

	fmt.Printf("\nInitializing map databases..\n")

	// Create or open DB
	auroraDB, err := store.NewMapDB("./db", common.SUPPORTED_MAPS.AURORA)
	if err != nil {
		log.Fatalf("Cannot initialize database for map '%s':\n%v", common.SUPPORTED_MAPS.AURORA, err)
	}

	// Deinfe all stores we want to exist on Aurora database.
	// If a store does not exist, it is created under ./db/aurora.
	store.DefineStore[oapi.ServerInfo](auroraDB, "server")
	store.DefineStore[oapi.TownInfo](auroraDB, "towns")
	store.DefineStore[oapi.NationInfo](auroraDB, "nations")
	store.DefineStore[oapi.EntityList](auroraDB, "entities")
	store.DefineStore[store.Alliance](auroraDB, "alliances")
	store.DefineStore[store.UserUsage](auroraDB, "usage-users")
	//store.AssignStoreToDB[map[string]any](auroraDB, "usage-leaderboard")

	fmt.Printf("\n\nEstablishing connection to Discord..\n")

	// Open WS connection to Discord.
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	// Wait for Ctrl+C or kill.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))

	// Since the `defer` keyword only works in successful exits,
	// closing explicitly here makes sure we always properly cleanup.
	if err := auroraDB.Close(); err != nil {
		log.Printf("Error closing DB: %v", err)
	}
	if err := s.Close(); err != nil {
		log.Printf("Error closing Discord session: %v", err)
	}
}
