package main

import (
	"context"
	"emcsrw/api/capi"
	"emcsrw/bot"
	"emcsrw/utils/config"
	"fmt"
	"log"
	"net/http"
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

func getBotToken() string {
	v, err := config.GetEnviroVar("BOT_TOKEN")
	if err != nil {
		log.Fatal(err)
	}

	// Don't rly need to parse since we already have string
	return v
}

func shouldServeAPI() bool {
	v, err := config.GetEnviroVar("ENABLE_API")
	if err != nil {
		if strings.Contains(err.Error(), "must be specified") {
			return false // By default, we don't want to serve if var is missing.
		}

		log.Fatal(err)
	}

	// String exists and not empty. Check it is a valid bool value
	parsed, err := config.ParseEnviroVar[bool](v)
	if err != nil {
		log.Fatal(err)
	}

	return parsed
}

func main() {
	// Start the bot with the token.
	loadEnv()

	fmt.Printf("Loaded ENV. Starting bot with %d threads.\n", runtime.GOMAXPROCS(-1))
	discord, auroraDB := bot.Run(getBotToken())

	var server *http.Server
	if shouldServeAPI() {
		mux, err := capi.NewMux(auroraDB)
		if err != nil {
			log.Println("failed to create api mux for aurora", err)
		}

		server = capi.Serve(mux)
	}

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

	// Gracefully shutdown HTTP server that serves the Custom API.
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("error shutting down HTTP server: %v", err)
		}
	}

	// Write every store to disk safely. Any store errs are combined into single error.
	if err := auroraDB.Flush(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
}
