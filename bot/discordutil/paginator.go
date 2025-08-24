package discordutil

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type Page struct {
	Embed *discordgo.MessageEmbed
}

type Paginator struct {
	Session     *discordgo.Session
	ChannelID   string
	UserID      string
	Pages       []Page
	CurrentPage int
	Timeout     time.Duration
	MessageID   string
	stopChan    chan struct{}
}

const fiveMin = 5 * time.Minute

func NewPaginator(s *discordgo.Session, channelID, userID string, pages []Page) *Paginator {
	return &Paginator{
		Session:     s,
		ChannelID:   channelID,
		UserID:      userID,
		Pages:       pages,
		CurrentPage: 0,
		Timeout:     fiveMin,
		stopChan:    make(chan struct{}),
	}
}
