package main

import (
	"context"
	"emcsrw/api/capi"
	"emcsrw/bot"
	"emcsrw/bot/scheduler"
	"emcsrw/bot/slashcommands"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/config"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
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

func main() {
	//#region Always runs no matter the subcommand
	if len(os.Args) < 2 {
		fmt.Println("ERROR | missing subcommand. Usage: go run . [register|bot|api]")
		return
	}

	config.LoadEnv()
	fmt.Println("DEBUG | Loaded .env into OS environment.")

	s, err := newSession(config.GetBotToken())
	if err != nil {
		fmt.Printf("\nFATAL | failed to create session:\n\t%s", err)
		os.Exit(67) // SIX SEVEEEEEEN!!!1!!1!!1
	}
	//#endregion

	subCmd := os.Args[1]
	if subCmd == "register" {
		slashcommands.SyncRemote(s, config.GetBotID(), "") // Empty str = register globally
	} else {
		activeMapDB := database.TryInit(shared.ACTIVE_MAP)

		switch subCmd {
		case "bot":
			unlock := lockProcess()
			defer unlock()

			startBot(s, activeMapDB)
		case "api":
			auroraDB := database.TryInit(shared.SUPPORTED_MAPS.AURORA)
			startAPI([]*database.Database{activeMapDB, auroraDB})
		default:
			fmt.Println("ERROR | unknown subcommand:", subCmd)
		}
	}
}

func newSession(token string) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return s, err
}

func startBot(s *discordgo.Session, activeMapDB *database.Database) {
	log.Printf("Starting bot with %d threads.\n", runtime.GOMAXPROCS(-1))

	// Init a scheduler that we can use to schedule tasks (ie. in OnReady)
	scheduler.Instance = scheduler.New()

	log.Println("Connecting to Discord gateway...")
	discord := bot.Connect(s)

	// Wait for Ctrl+C or kill.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))

	// Gracefully stop scheduled tasks by waiting for all to finish.
	msg := scheduler.Instance.Shutdown(30 * time.Second)
	log.Println("[Scheduler]: " + msg)

	//#region Cleanup
	// Since the `defer` keyword only works in successful exits,
	// closing explicitly here makes sure we always properly cleanup.
	if err := discord.Close(); err != nil {
		log.Printf("error closing Discord session: %v", err)
	}

	// Write every store to disk safely. Any store errs are combined into single error.
	if err := activeMapDB.Flush(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
	//endregion
}

func startAPI(mdbs []*database.Database) {
	port := config.GetApiPort()
	if capi.IsRunning(port) {
		log.Fatalf("Custom API server already listening on :%d", port)
		return
	}

	mux, err := capi.NewMux(mdbs)
	if err != nil {
		log.Fatalf("failed to start Custom API. failed to init mux.\n%s", err)
	}

	server := capi.Serve(mux, config.GetApiPort())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	fmt.Printf("\nshutting down Custom API with signal: %s\n", strings.ToUpper(sig.String()))

	// Gracefully shutdown HTTP server that serves the Custom API.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("error shutting down Custom API: %v", err)
	}
}
