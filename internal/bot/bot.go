package bot

import (
	"emcsrw/internal/bot/events"
	"emcsrw/internal/bot/scheduler"
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/utils/config"
	"emcsrw/pkg/utils/logutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ==============================!! WARNING !! ==============================
// If some of these intents are not granted to you by Discord, the bot may not start correctly.
// Remove all intents you do not have access to or require and restart the bot again.
//
// See the Privileged Intents section: https://docs.discord.com/developers/events/gateway#gateway-intents
// ==========================================================================

// var DM_INTENTS = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions
var ALL_INTENTS = discordgo.IntentMessageContent | GUILD_INTENTS
var GUILD_INTENTS = discordgo.IntentGuilds |
	discordgo.IntentGuildMessages |
	discordgo.IntentGuildMessageReactions

// Specifys the events we want to handle.
// Make sure we actually listen and respond to them once specified to avoid errors!
//
// https://discord.com/developers/docs/events/gateway-events#receive-events
var EVENT_HANDLERS = [...]any{
	events.OnReady,
	events.OnDisconnect,
	events.OnApplicationCommandInteractionCreate,
	events.OnModalSubmitInteractionCreate,
	events.OnSelectMenuInteractionCreate,
	events.OnButtonInteractionCreate,
	events.OnAutocompleteInteractionCreate,
}

// Uses session s to open a WebSocket connection to the Discord gateway once all necessary event handlers
// have been registered via EVENT_HANDLERS - these are called when their respective event is fired by the
// Discord websocket/gateway API and the function signature matches.
func ConnectGateway(s *discordgo.Session) *discordgo.Session {
	s.Identify.Intents = ALL_INTENTS
	s.SyncEvents = false // Run handlers in a goroutine to prevent a command waiting on another user's command.
	s.ShouldReconnectOnError = true
	s.LogLevel = discordgo.LogError // Keep commented unless required to diagnose Discord issues.
	for _, h := range EVENT_HANDLERS {
		s.AddHandler(h)
	}

	// Open websocket connection to Discord gateway.
	err := s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	logutil.Logln(logutil.BLUE, "Established WS connection to Discord.")
	return s
}

func DisconnectGateway(s *discordgo.Session) {
	done := make(chan error, 1)
	go func() {
		done <- s.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			logutil.Logf(logutil.RED, "error closing Discord session: %v", err)
		}
	case <-time.After(5 * time.Second):
		logutil.Println(logutil.RED, "WARN | Call to close Discord session timed out. Continuing shutdown..")
	}
}

// Start the bot process (db init, scheduler init, discord connection, etc.) and block
// until a termination signal is received at which point a graceful shutdown will occur.
func Start(s *discordgo.Session) {
	activeMapDB := database.TryInit(shared.ACTIVE_MAP)

	logutil.Logf(logutil.BLUE, "Starting bot with %d threads.", runtime.GOMAXPROCS(-1))

	// Init a scheduler that we can use to schedule tasks (ie. in OnReady)
	scheduler.Instance = scheduler.New()

	logutil.Logln(logutil.BLUE, "Connecting to Discord gateway...")
	ConnectGateway(s)

	// ctx, stopSSE := context.WithCancel(context.Background())
	// go func() {
	// 	oapiAuthKey := os.Getenv("OAPI_AUTH_KEY")
	// 	if err := oapi.ListenToSSE(ctx, oapi.GLOBAL_EVENTS[:], oapiAuthKey); err != nil {
	// 		log.Printf("SSE stopped: %v", err)
	// 	}
	// }()

	//#region Handle graceful shutdown upon a termination signal.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM) // Interrupt = Ctrl+C | SIGHUP = tmux kill | SIGTERM = kill
	//defer signal.Stop(c) // Not needed right now. the process exits after Start returns.

	sig := <-c
	events.ShuttingDown.Store(true)

	logutil.Printf(logutil.YELLOW, "\n\nReceived signal: %s", strings.ToUpper(sig.String()))
	Shutdown(s, activeMapDB)
	//#endregion
}

// Gracefully shutdown the bot by shutting down the data scheduler, closing the websocket connection
// of the current Discord session and flushing the current state of the active map database to disk.
func Shutdown(s *discordgo.Session, activeMapDB *database.Database) {
	//stopSSE()

	timeout := 60
	if t, err := config.ParseEnviroVar[int]("SHUTDOWN_TIMEOUT_SEC"); err == nil {
		timeout = t
	}
	logutil.Printf(logutil.YELLOW, "\nAttempting graceful shutdown. Waiting up to %d seconds or until all tasks finish.\n", timeout)

	// Begin stopping scheduler tasks and wait for current ones to finish.
	logutil.Println(logutil.FAINT, "DEBUG | Shutdown: Scheduler")
	msg := scheduler.Instance.Shutdown(time.Duration(timeout) * time.Second)
	logutil.Logln(logutil.BLUE, "[Scheduler]: "+msg)

	// Close the existing WS connection with Discord.
	logutil.Println(logutil.FAINT, "DEBUG | Shutdown: Discord")
	DisconnectGateway(s)

	// Write every store to disk safely. All store errs during this are combined into single error.
	logutil.Println(logutil.FAINT, "DEBUG | Shutdown: DB")
	if err := activeMapDB.Flush(); err != nil {
		logutil.Logf(logutil.RED, "error flushing DB: %v", err)
	}
}
