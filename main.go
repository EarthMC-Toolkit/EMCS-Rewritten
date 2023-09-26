package main

import (
	"emcsrw/bot"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil { log.Fatal(err) }
}

func getToken() string {
	token, _ := os.LookupEnv("BOT_TOKEN")
	if token == "" {
		log.Fatal("Could use Discord token! Make sure it's set with 'BOT_TOKEN'")
	}

	return token
}

func main() {
	// bot.Test()
	// return

	// Start the bot with the token.
	loadEnv()
	bot.Run(getToken())
}