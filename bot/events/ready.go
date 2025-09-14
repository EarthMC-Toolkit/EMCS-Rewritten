package events

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/slashcommands"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
)

// Leave empty to register commands globally
const guildID = ""

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Logged in as: %s\n\n", s.State.User.Username)

	// TODO: Combine these into a single pass if possible. Something like `SyncSlashCommands()`.
	CleanupOldCommands(s)
	RegisterSlashCommands(s)

	fmt.Printf("\n\n")

	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)

	//api.QueryAndSaveTowns()
	scheduleTask(func() {
		QueryAndSaveServerInfo(db)
	}, true, 60*time.Second)
}

// Deletes any commands existing on the remote (what discord has registered) as long as they
// aren't currently registered on the local side via slashcommands.All()
func CleanupOldCommands(s *discordgo.Session) {
	discordCmds, err := s.ApplicationCommands(s.State.User.ID, guildID) // Get commands discord has registered.
	if err != nil {
		fmt.Printf("Cannot clean up old commands. Failed to query discord: %v", err)
		return
	}

	for _, cmd := range discordCmds {
		_, ok := slashcommands.All()[cmd.Name] // Check this cmd still exists in ones we registered.
		if !ok {
			// Doesn't exist, must be old. Delete dat shit
			err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
			if err != nil {
				fmt.Printf("Failed to delete old command %s: %v", cmd.Name, err)
				continue
			}

			fmt.Printf("Deleted stale remote command: %s", cmd.Name)
		}
	}
}

func RegisterSlashCommands(s *discordgo.Session) {
	for _, cmd := range slashcommands.All() {
		fmt.Printf("Registering slash command '%s'\n", cmd.Name())

		_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, slashcommands.ToApplicationCommand(cmd))
		if err != nil {
			fmt.Printf("Failed to register slash command '%v': %v\n", cmd.Name(), err)
		}
	}
}

func scheduleTask(task func(), runInitial bool, interval time.Duration) chan struct{} {
	if runInitial {
		task()
	}

	stop := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			task()
		}
	}()

	return stop
}

func QueryAndSaveAllTowns(mapDB *badger.DB) {
	towns, err := api.QueryAllTowns()
	if err != nil {
		fmt.Printf("error putting towns into db at %s:\n%v", mapDB.Opts().Dir, err)
		return
	}

	data, err := json.Marshal(towns)
	if err != nil {
		fmt.Printf("error putting towns into db at %s:\n%v", mapDB.Opts().Dir, err)
		return
	}

	database.PutInsensitive(mapDB, "towns", data)
	fmt.Printf("put towns into db at %s\n", mapDB.Opts().Dir)
}

func QueryAndSaveServerInfo(mapDB *badger.DB) {
	info, err := oapi.QueryServer()
	if err != nil {
		fmt.Printf("error putting server info into db at %s:\n%v", mapDB.Opts().Dir, err)
		return
	}

	data, err := json.Marshal(info)
	if err != nil {
		fmt.Printf("error putting server info into db at %s:\n%v", mapDB.Opts().Dir, err)
		return
	}

	// NOTE: Consider putting VP and statistics seperately. Is this optimization worth it?
	database.PutInsensitive(mapDB, "serverinfo", data)
	fmt.Printf("put server info into db at %s\n", mapDB.Opts().Dir)
}
