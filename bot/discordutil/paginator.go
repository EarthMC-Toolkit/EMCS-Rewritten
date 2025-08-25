package discordutil

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const fiveMin = 5 * time.Minute

type Paginator struct {
	Session     *discordgo.Session
	ChannelID   string
	UserID      string
	CurrentPage int
	Timeout     time.Duration
	MessageID   string
	stopChan    chan struct{}
}

type InteractionPageFunc func(page int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	PageFunc InteractionPageFunc
}

func NewInteractionPaginator(s *discordgo.Session, channelID, userID string, pageFunc InteractionPageFunc) *InteractionPaginator {
	return &InteractionPaginator{
		PageFunc: pageFunc,
		Paginator: &Paginator{
			Session:     s,
			ChannelID:   channelID,
			UserID:      userID,
			CurrentPage: 0,
			Timeout:     fiveMin,
			stopChan:    make(chan struct{}),
		},
	}
}
