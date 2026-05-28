package bot

import (
	"emcsrw/bot/events"
	"emcsrw/bot/scheduler"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/logutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// var DM_INTENTS = dgo.IntentDirectMessages | dgo.IntentDirectMessageReactions

// !! WARNING !!
// If you do not have some of these intents, the bot may not start correctly.
// Remove any you do not require and restart the bot again.
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
	events.OnApplicationCommandInteractionCreate,
	events.OnModalSubmitInteractionCreate,
	events.OnSelectMenuInteractionCreate,
	events.OnButtonInteractionCreate,
	events.OnAutocompleteInteractionCreate,
}

// Uses session s to open a WebSocket connection to the Discord gateway once all necessary event handlers
// have been registered via EVENT_HANDLERS - these are called when their respective event is fired by the
// Discord websocket/gateway API and the function signature matches.
func Connect(s *discordgo.Session) *discordgo.Session {
	s.Identify.Intents = ALL_INTENTS
	s.SyncEvents = false // Run handlers in a goroutine to prevent a command waiting on another user's command.

	for _, eh := range EVENT_HANDLERS {
		s.AddHandler(eh)
	}

	// Open websocket connection to Discord gateway.
	err := s.Open()
	if err != nil {
		log.Fatal("Cannot open Discord session: ", err)
	}

	log.Println("Established connection to Discord.")
	return s
}

// Start the bot process (db init, scheduler init, discord connection, etc.) and block
// until a termination signal is received at which point a graceful shutdown will occur.
func Start(s *discordgo.Session) {
	activeMapDB := database.TryInit(shared.ACTIVE_MAP)

	log.Printf("Starting bot with %d threads.\n", runtime.GOMAXPROCS(-1))

	// Init a scheduler that we can use to schedule tasks (ie. in OnReady)
	scheduler.Instance = scheduler.New()

	log.Println("Connecting to Discord gateway...")
	Connect(s)

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
	sig := <-c

	logutil.Printf(logutil.YELLOW, "\nShutting down bot with signal: %s\n", strings.ToUpper(sig.String()))
	Shutdown(s, activeMapDB)
	//#endregion
}

// Gracefully shutdown the bot by shutting down the data scheduler, closing the websocket connection
// of the current Discord session and flushing the current state of the active map database to disk.
func Shutdown(s *discordgo.Session, activeMapDB *database.Database) {
	//stopSSE()

	msg := scheduler.Instance.Shutdown(30 * time.Second)
	logutil.Println(logutil.YELLOW, "[Scheduler]: "+msg)

	// Since the `defer` keyword only works in successful exits,
	// closing explicitly here makes sure we always properly cleanup.
	if err := s.Close(); err != nil {
		logutil.Logf(logutil.RED, "error closing Discord session: %v", err)
	}

	// Write every store to disk safely. Any store errs during this are combined into single error.
	if err := activeMapDB.Flush(); err != nil {
		logutil.Logf(logutil.RED, "error closing DB: %v", err)
	}
}
