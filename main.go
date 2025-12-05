package main

import (
	"context"
	"emcsrw/api/capi"
	"emcsrw/bot"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
}

func getToken() string {
	token, _ := os.LookupEnv("BOT_TOKEN")
	if token == "" {
		log.Fatal("Could use Discord token! Make sure it's set with 'BOT_TOKEN'")
	}

	return token
}

func main() {
	// Start the bot with the token.
	loadEnv()

	fmt.Printf("Loaded ENV. Starting bot with %d threads.\n", runtime.GOMAXPROCS(-1))
	discord, auroraDB := bot.Run(getToken())

	mux, err := capi.NewMux(auroraDB)
	if err != nil {
		log.Println("failed to create api mux for aurora", err)
	}
	s := capi.Serve(mux)

	// Wait for Ctrl+C or kill.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))

	// Since the `defer` keyword only works in successful exits,
	// closing explicitly here makes sure we always properly cleanup.
	if err := discord.Close(); err != nil {
		log.Printf("error closing Discord session: %v", err)
	}

	// Gracefully shutdown HTTP server for Custom API.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error shutting down HTTP server: %v", err)
	}

	// Write every store to disk safely. Any store errs are combined into single error.
	if err := auroraDB.Flush(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
}
