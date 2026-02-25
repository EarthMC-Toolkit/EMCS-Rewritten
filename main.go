package main

import (
	"context"
	"emcsrw/api/capi"
	"emcsrw/bot"
	"emcsrw/bot/slashcommands"
	"emcsrw/shared"
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

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/rogpeppe/go-internal/lockedfile"
)

const LOCK_FPATH = "/tmp/emcsrw.lock"

// Creates a lock file and returns a handle to unlock it (remove the file).
func lockProcess() func() error {
	lock, err := lockedfile.OpenFile(LOCK_FPATH, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("DEBUG | Acquired lock from ~" + LOCK_FPATH)
	return func() error {
		err := lock.Close()
		if err != nil {
			fmt.Println("ERROR | Failed to close lock at ~" + LOCK_FPATH)
			return err
		}

		fmt.Println("DEBUG | Removed lock from ~" + LOCK_FPATH)
		return nil
	}
}

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

func getBotID() string {
	v, err := config.GetEnviroVar("BOT_APP_ID")
	if err != nil {
		log.Fatal(err)
	}

	return v
}

func getApiPort() uint {
	fail := func(reason string) uint {
		fmt.Printf("\nERROR | Custom API port defaulted to 7777. Reason:\n\t%s\n", reason)
		return 7777
	}

	v, err := config.GetEnviroVar("API_PORT")
	if err != nil {
		return fail(err.Error())
	}

	port, err := config.ParseEnviroVar[uint](v)
	if err != nil {
		return fail(err.Error())
	}

	switch port {
	case 80, 443:
		return port // Allow HTTP and HTTPS default ports
	default:
		if port < 1024 || port > 49150 {
			return fail("environment variable API_PORT must be 80, 443 or in range 1024-49150")
		}
	}

	return port
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
	//#region Always runs no matter the subcommand
	unlock := lockProcess()
	defer unlock()

	if len(os.Args) < 2 {
		fmt.Println("ERROR | missing subcommand. Usage: go run . [register|start]")
		return
	}

	loadEnv()
	fmt.Println("DEBUG | Loaded .env into OS environment.")

	s, err := newSession(getBotToken())
	if err != nil {
		fmt.Printf("\nFATAL | failed to create session:\n\t%s", err)
		os.Exit(67) // SIX SEVEEEEEEN!!!1!!1!!1
	}
	//#endregion

	switch os.Args[1] {
	case "register":
		slashcommands.SyncRemote(s, getBotID(), "") // Empty str = register globally
	case "start":
		startBot(s)
	default:
		fmt.Println("ERROR | unknown subcommand:", os.Args[1])
	}
}

func newSession(token string) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return s, err
}

func startBot(s *discordgo.Session) {
	log.Printf("Starting bot with %d threads.\n", runtime.GOMAXPROCS(-1))

	activeMapDB := bot.InitDB(shared.ACTIVE_MAP)
	log.Printf("Initialized databases: %s\n", strings.Join([]string{shared.ACTIVE_MAP}, ","))

	server := &http.Server{}

	if shouldServeAPI() {
		mux, err := capi.NewMux(activeMapDB)
		if err != nil {
			fmt.Printf("failed to create api mux for %s:\n%s", activeMapDB, err)
		}

		server = capi.Serve(mux, getApiPort())
	}

	log.Println("Connecting to Discord gateway...")
	var discord = bot.Connect(s)

	// Wait for Ctrl+C or kill.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))

	//#region Cleanup
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
	if err := activeMapDB.Flush(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
	//endregion
}
