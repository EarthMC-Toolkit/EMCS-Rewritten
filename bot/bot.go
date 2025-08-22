package bot

import (
	"emcsrw/bot/events"
	"emcsrw/bot/slashcommands"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	dgo "github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Leave empty to register commands globally
const testGuildID = ""

//const RED, YELLOW int = 8858420, 15844367

var guildIntents = dgo.IntentGuilds | dgo.IntentGuildMessages | dgo.IntentGuildMessageReactions

//var dmIntents = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

func Run(botToken string) {
	// Initialize a Discord Session
	s, err := dgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}

	// Register handler funcs for gateway events.
	// https://discord.com/developers/docs/events/gateway-events#receive-events
	s.AddHandler(events.OnInteractionCreate)
	s.AddHandler(events.OnReady)

	s.Identify.Intents = dgo.IntentMessageContent | guildIntents

	// Open WS connection to Discord
	err = s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}
	defer s.Close()

	RegisterSlashCommands(s)

	// Run until code is terminated
	fmt.Printf("Bot running...\n")

	// Wait for Ctrl+C (exit)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))
}

func RegisterSlashCommands(s *dgo.Session) {
	// Register commands
	slashCmds := slashcommands.All()
	cmdsAmt := len(slashCmds)

	fmt.Println("Registering " + strconv.Itoa(cmdsAmt) + " slash commands.")
	for _, cmd := range slashCmds {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, testGuildID, slashcommands.ToApplicationCommand(cmd))
		if err != nil {
			fmt.Printf("Cannot create '%v' command: %v\n", cmd.Name(), err)
		}
	}
}

// func SendComplex(discord *dgo.Session, message *dgo.MessageCreate, embed *dgo.MessageSend) {
// 	_, err := discord.ChannelMessageSendComplex(message.ChannelID, embed)
// 	if err != nil {
// 		log.Error(err)
// 	}
// }
