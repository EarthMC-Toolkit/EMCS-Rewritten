package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Logged in as", r.User.Username)
}
