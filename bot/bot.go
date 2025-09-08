package bot

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/events"
	"fmt"
	"os"
	"os/signal"
	"strings"

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

	// Register funcs that handle specific gateway events.
	// https://discord.com/developers/docs/events/gateway-events#receive-events
	s.AddHandler(events.OnReady)
	s.AddHandler(events.OnInteractionCreateApplicationCommand) // Slash cmds
	s.AddHandler(events.OnInteractionCreateMessageComponent)   // Buttons, rows, select menus

	s.Identify.Intents = dgo.IntentMessageContent | guildIntents

	// Run until code is terminated
	fmt.Printf("\nEstablishing Discord connection..\n")

	// Open WS connection to Discord
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}
	defer s.Close()

	// Create or open badger DB
	auroraDB, err := database.InitMapDB(common.SUPPORTED_MAPS.AURORA)
	if err != nil {
		log.Fatalf("Cannot initialize database for map '%s':\n%v", common.SUPPORTED_MAPS.AURORA, err)
	}
	defer auroraDB.Close()

	// Wait for Ctrl+C (exit)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))
}
