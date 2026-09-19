package main

import (
	"emcsrw/internal/bot"
	"emcsrw/internal/bot/slashcommands"
	"emcsrw/pkg/api/capi"
	"emcsrw/pkg/config"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/flock"
	"github.com/samber/lo"
)

// The cross platform path to the lock file used to prevent multiple instances of the bot from running at the same time.
//   - Unix: ~/tmp/emcsrw.lock
//   - Windows: C:\Users\<user>\AppData\Local\Temp\emcsrw.lock
var lockPath = filepath.Join(os.TempDir(), "emcsrw.lock")

// The cross platform path to the log file used to log Discord events and errors for diagnostic purposes.
//   - Unix: ~/tmp/emcsrw.log
//   - Windows: C:\Users\<user>\AppData\Local\Temp\emcsrw.log
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

// Runs bot pre-setup required before it can run.
// Loads env and ultimately initializing a new discordgo/Discord session.
//
// Any errors returned from this func should ALWAYS indicate a fatal shutdown (enforced in main).
func setup() (*discordgo.Session, error) {
	// Load vars from appropriate env file into current OS environment.
	if err := config.LoadEnv(true); err != nil {
		return nil, err
	}
	logutil.DebugLogEnabled, _ = config.ParseEnviroVar[bool]("ENABLE_DEBUG_LOG")
	logutil.Println(logutil.FAINT, "DEBUG | Loaded .env into OS environment.")

	// Make a Discord session using configured bot token from loaded env.
	tkn, err := config.GetEnviroVar("BOT_TOKEN")
	if err != nil {
		return nil, err
	}
	s, err := discordgo.New("Bot " + tkn)
	if err != nil {
		return nil, err
	}
	logutil.Println(logutil.FAINT, "DEBUG | Discord session created.")

	return s, nil
}

func execCliSubcmd(subCmd string, s *discordgo.Session) error {
	switch subCmd {
	case "bot":
		if err := logutil.InitFile(logPath); err != nil {
			return fmt.Errorf("Failed to init log file at %s: %s", logPath, err)
		}

		s.LogLevel = discordgo.LogError
		discordgo.Logger = func(msgL, caller int, format string, a ...any) {
			logutil.FileLog.Printf("DISCORDGO | [DG%d] %s\n", msgL, fmt.Sprintf(format, a...))
		}

		bot.Start(s)
	case "api":
		capi.Start()
	case "register", "sync":
		appID, err := config.GetEnviroVar("BOT_APP_ID")
		if err != nil {
			return err
		}

		slashcommands.SyncRemote(s, appID, "")
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		logutil.Exit(1, "ERR | missing subcommand. Usage: go run . [sync|bot|api]")
	}

	subCmd := os.Args[1]
	if !lo.Contains([]string{"bot", "api", "register", "sync"}, subCmd) {
		logutil.Exit(1, "ERR | unknown subcommand:", subCmd)
	}

	unlock := func() error { return nil }
	if subCmd == "bot" {
		var err error
		unlock, err = lockProcess()
		if err != nil {
			logutil.Exit(1, "ERR |", err)
		}
		defer unlock()
	}

	s, err := setup()
	if err != nil {
		unlock()
		logutil.Fatalf(67, "FATAL | failed before executing subcommand '%s'. error during setup:\n\t%s\n", subCmd, err)
	}

	if err := execCliSubcmd(subCmd, s); err != nil {
		unlock()
		logutil.Fatalf(67, "FATAL | error executing subcommand '%s':\n\t%s\n", subCmd, err)
	}
}
