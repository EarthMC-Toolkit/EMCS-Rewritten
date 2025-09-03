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
	CurrentPage *int
	Timeout     time.Duration
	MessageID   string
	stopChan    chan struct{}
}

type InteractionPageFunc func(page int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	PageFunc InteractionPageFunc
}

func NewInteractionPaginator(s *discordgo.Session, i *discordgo.Interaction, pageFunc InteractionPageFunc) *InteractionPaginator {
	author := UserFromInteraction(i)
	initPage := 0

	return &InteractionPaginator{
		PageFunc: pageFunc,
		Paginator: &Paginator{
			Session:     s,
			ChannelID:   i.ChannelID,
			UserID:      author.ID,
			CurrentPage: &initPage,
			Timeout:     fiveMin,
			stopChan:    make(chan struct{}),
		},
	}
}

// Attempts to get the username from an interaction.
//
// Regular `User` is only filled for a DM, so this func uses guild-specific `Member.User` otherwise.
func UserFromInteraction(i *discordgo.Interaction) *discordgo.User {
	if i.User != nil {
		return i.User
	}

	return i.Member.User
}
