package bot

import (
	"emcsrw/bot/slashcommands"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	dgo "github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const RED, YELLOW int = 8858420, 15844367

var integrationTypes = []dgo.ApplicationIntegrationType{dgo.ApplicationIntegrationUserInstall} // 0 for Guild, 1 for User
var contexts = []dgo.InteractionContextType{dgo.InteractionContextGuild}                       // 0 for Guilds, 2 for DMs, 3 for Private Channels

var guildIntents = dgo.IntentGuilds | dgo.IntentGuildMessages | dgo.IntentGuildMessageReactions

//var dmIntents = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

func Run(botToken string) {
	// Create new Discord Session
	discord, err := dgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(interactionCreate)
	discord.Identify.Intents = dgo.IntentMessageContent | guildIntents

	// Open WS connection to Discord
	discord.Open()
	defer discord.Close()

	// Register commands
	cmds := slashcommands.All()
	cmdsAmt := len(cmds)

	fmt.Println("Registering " + strconv.Itoa(cmdsAmt) + " slash commands...")
	for _, cmd := range slashcommands.All() {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", &dgo.ApplicationCommand{
			Name:             cmd.Name(),
			Description:      cmd.Description(),
			Options:          cmd.Options(),
			IntegrationTypes: &integrationTypes,
			Contexts:         &contexts,
		})

		if err != nil {
			fmt.Printf("Cannot create '%v' command: %v\n", cmd.Name(), err)
		}
	}

	// Run until code is terminated
	fmt.Println("Bot running...")

	// Wait for Ctrl+C (exit)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	fmt.Println("Shutting down gracefully...")
}

func SendComplex(discord *dgo.Session, message *dgo.MessageCreate, embed *dgo.MessageSend) {
	_, err := discord.ChannelMessageSendComplex(message.ChannelID, embed)
	if err != nil {
		log.Error(err)
	}
}

// TODO: Implement slash cmds
