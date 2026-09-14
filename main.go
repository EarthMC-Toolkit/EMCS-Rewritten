package main

import (
	"emcsrw/internal/bot"
	"emcsrw/internal/bot/slashcommands"
	"emcsrw/pkg/api/capi"
	"emcsrw/pkg/utils/config"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/flock"
)

// The cross platform path to the lock file used to prevent multiple instances of the bot from running at the same time.
//   - Unix: ~/tmp/emcsrw.lock
//   - Windows: C:\Users\<user>\AppData\Local\Temp\emcsrw.lock
var lockPath = filepath.Join(os.TempDir(), "emcsrw.lock")
var logPath = filepath.Join(os.TempDir(), "emcsrw.log")

// Attempts to acquire an exclusive process lock.
// Returns an unlock function if successful, or an error if another instance already holds the lock.
func lockProcess() (func() error, error) {
	lock := flock.New(lockPath)
	if locked, err := lock.TryLock(); err != nil {
		return nil, err
	} else if !locked {
		return nil, fmt.Errorf("another instance of EMCS is already running")
	}

	logutil.Println(logutil.FAINT, "DEBUG | Acquired process lock")
	return func() error {
		err := lock.Unlock()
		if err == nil {
			logutil.Println(logutil.FAINT, "DEBUG | Released process lock")
		}
		return err
	}, nil
}

func main() {
	//#region Always runs no matter the subcommand
	if len(os.Args) < 2 {
		logutil.Println(logutil.RED, "ERR | missing subcommand. Usage: go run . [sync|bot|api]")
		return
	}

	subCmd := os.Args[1]
	if subCmd == "bot" {
		unlock, err := lockProcess()
		if err != nil {
			logutil.Println(logutil.RED, "ERR |", err)
			os.Exit(1)
		}
		defer unlock() // should run just before main returns and the process exits.
	}

	config.LoadEnv()
	logutil.Println(logutil.FAINT, "DEBUG | Loaded .env into OS environment.")

	s, err := newSession(config.GetBotToken())
	if err != nil {
		logutil.Printf(logutil.RED, "\nFATAL | Failed to create Discord session:\n\t%s", err)
		os.Exit(67) // SIX SEVEEEEEEN!!!1!!1!!1
	}
	//#endregion

	switch subCmd {
	case "bot":
		if err := logutil.InitFile(logPath); err != nil {
			logutil.Println(logutil.RED, "ERR | Failed to initialize log file at ", logPath, ":", err)
			return
		}

		s.LogLevel = discordgo.LogError // Keep commented unless required to diagnose Discord issues.
		discordgo.Logger = func(msgL, caller int, format string, a ...any) {
			logutil.FileLog.Printf("DISCORDGO | [DG%d] %s\n", msgL, fmt.Sprintf(format, a...))
		}

		bot.Start(s)
	case "api":
		capi.Start()
	case "register", "sync":
		slashcommands.SyncRemote(s, config.GetBotID(), "") // Empty str = register commands globally
	default:
		logutil.Println(logutil.RED, "ERR | unknown subcommand:", subCmd)
	}
}

func newSession(token string) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	logutil.Println(logutil.FAINT, "DEBUG | Discord session created.")
	return s, err
}
