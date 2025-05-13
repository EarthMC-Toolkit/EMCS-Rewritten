package bot

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

const RED, YELLOW int = 8858420, 15844367

func Run(botToken string) {
	// Create new Discord Session
	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}

	//discord.AddHandler(messageCreate)

	// Open session
	discord.Open()
	defer discord.Close()

	// Run until code is terminated
	fmt.Println("Bot running...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func EmbedField(name string, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

// TODO: Implement slash cmds
