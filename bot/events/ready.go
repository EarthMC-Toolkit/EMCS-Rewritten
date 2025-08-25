package events

import (
	"emcsrw/bot/slashcommands"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Leave empty to register commands globally
const testGuildID = ""

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Logged in as: %s\n\n", s.State.User.Username)

	RegisterSlashCommands(s)

	fmt.Println()

	scheduleTask(func() { fmt.Println("hi") }, 5*time.Second)
}

func RegisterSlashCommands(s *discordgo.Session) {
	for _, cmd := range slashcommands.All() {
		fmt.Printf("Registering slash command: %s.\n", cmd.Name())

		_, err := s.ApplicationCommandCreate(s.State.User.ID, testGuildID, slashcommands.ToApplicationCommand(cmd))
		if err != nil {
			fmt.Printf("Cannot create '%v' command: %v\n", cmd.Name(), err)
		}
	}
}

func scheduleTask(fn func(), interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fn()
			case <-stop:
				return
			}
		}
	}()

	return stop
}
