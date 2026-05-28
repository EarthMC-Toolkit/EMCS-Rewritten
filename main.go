package main

import (
	"emcsrw/api/capi"
	"emcsrw/bot"
	"emcsrw/bot/slashcommands"
	"emcsrw/utils/config"
	"emcsrw/utils/logutil"
	"log"
	"os"

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

	logutil.Println(logutil.HIDDEN, "DEBUG | Acquired lock from ~"+LOCK_FPATH)
	return func() error {
		err := lock.Close()
		if err != nil {
			logutil.Println(logutil.RED, "ERR | Failed to close lock at ~"+LOCK_FPATH)
			return err
		}

		logutil.Println(logutil.HIDDEN, "DEBUG | Removed lock from ~"+LOCK_FPATH)
		return nil
	}
}

func main() {
	//#region Always runs no matter the subcommand
	if len(os.Args) < 2 {
		logutil.Println(logutil.RED, "ERR | missing subcommand. Usage: go run . [register|bot|api]")
		return
	}

	config.LoadEnv()
	logutil.Println(logutil.HIDDEN, "DEBUG | Loaded .env into OS environment.")

	s, err := newSession(config.GetBotToken())
	if err != nil {
		logutil.Printf(logutil.RED, "\nFATAL | failed to create session:\n\t%s", err)
		os.Exit(67) // SIX SEVEEEEEEN!!!1!!1!!1
	}
	//#endregion

	subCmd := os.Args[1]
	switch subCmd {
	case "register":
		slashcommands.SyncRemote(s, config.GetBotID(), "") // Empty str = register globally
	case "bot":
		unlock := lockProcess()
		defer unlock()

		bot.Start(s)
	case "api":
		capi.Start()
	default:
		logutil.Println(logutil.RED, "ERR | unknown subcommand:", subCmd)
	}
}

func newSession(token string) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return s, err
}
