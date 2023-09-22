package main

import (
	"emcs-rewritten/bot"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Grab env vars
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Could use Discord token! Make sure it's set with 'BOT_TOKEN'")
	}

	// Start up the bot
	bot.BotToken = os.Getenv("BOT_TOKEN")
	bot.Run()
}