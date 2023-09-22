package main

import (
	"emcs-rewritten/bot"
	"log"
	"os"
)

func main() {
	// Grab env vars
	botToken, ok := os.LookupEnv("BOT_TOKEN")

	if !ok {
		log.Fatal("Could not find Discord token! Must be set using: BOT_TOKEN")
	}

	// Start up the bot
	bot.BotToken = botToken
	bot.Run()
}