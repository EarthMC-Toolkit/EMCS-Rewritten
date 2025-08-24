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

type Page struct {
	Embed *discordgo.MessageEmbed
}

type CachedPaginator struct {
	*Paginator
	Pages []Page
}

func NewCachedPaginator(s *discordgo.Session, channelID, userID string, pages []Page) *CachedPaginator {
	return &CachedPaginator{
		Pages: pages,
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

type PageFunc func(page int) *discordgo.MessageEmbed

type LazyPaginator struct {
	*Paginator
	PageFunc PageFunc
}

func NewLazyPaginator(s *discordgo.Session, channelID, userID string, pageFunc PageFunc) *LazyPaginator {
	return &LazyPaginator{
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
