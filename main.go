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
	unlock := lockProcess()
	defer unlock()

	if len(os.Args) < 2 {
		fmt.Println("ERROR | missing subcommand. Usage: go run . [register|start]")
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

	switch os.Args[1] {
	case "register":
		slashcommands.SyncRemote(s, config.GetBotID(), "") // Empty str = register globally
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
	if config.ShouldServeAPI() {
		mux, err := capi.NewMux(activeMapDB)
		if err != nil {
			fmt.Printf("failed to create api mux for %s:\n%s", activeMapDB, err)
		}

		server = capi.Serve(mux, config.GetApiPort())
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
