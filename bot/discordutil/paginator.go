package discordutil

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const fiveMin = 5 * time.Minute

type Paginator struct {
	session     *discordgo.Session
	channelID   string
	userID      string
	currentPage *int
	pageLen     int
	totalPages  int
	timeout     time.Duration
	stopChan    chan struct{}
}

func (p *Paginator) TotalPages() int {
	return p.totalPages
}

type InteractionPageFunc func(curPage, pageLen int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	interaction *discordgo.Interaction
	cache       map[int]*discordgo.InteractionResponseData
	PageFunc    InteractionPageFunc
}

func NewInteractionPaginator(s *discordgo.Session, i *discordgo.Interaction, totalItems, pageLen int) *InteractionPaginator {
	author := UserFromInteraction(i)

	initPage := 0
	totalPages := (totalItems + pageLen - 1) / pageLen // round to next largest int (ceil)

	return &InteractionPaginator{
		interaction: i,
		cache:       make(map[int]*discordgo.InteractionResponseData),
		Paginator: &Paginator{
			session:     s,
			channelID:   i.ChannelID,
			userID:      author.ID,
			currentPage: &initPage,
			totalPages:  totalPages,
			timeout:     fiveMin,
			stopChan:    make(chan struct{}),
		},
	}
}

func (p *InteractionPaginator) NewNavigationButtonRow() discordgo.ActionsRow {
	curPage := *p.currentPage

	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "<<",
				CustomID: "first",
				Style:    discordgo.PrimaryButton,
				Disabled: curPage == 0,
			},
			discordgo.Button{
				Label:    "<",
				CustomID: "prev",
				Style:    discordgo.SuccessButton,
				Disabled: curPage == 0,
			},
			discordgo.Button{
				Label:    ">",
				CustomID: "next",
				Style:    discordgo.SuccessButton,
				Disabled: curPage == p.TotalPages()-1,
			},
			discordgo.Button{
				Label:    ">>",
				CustomID: "last",
				Style:    discordgo.PrimaryButton,
				Disabled: curPage == p.TotalPages()-1,
			},
		},
	}
}

func (p *InteractionPaginator) getPageData(page int) *discordgo.InteractionResponseData {
	data, ok := p.cache[page]
	if !ok {
		// Page not already cached, render page and cache.
		data = &discordgo.InteractionResponseData{}
		p.PageFunc(page, p.pageLen, data)
		p.cache[page] = data
	}

	return data
}

func (p *InteractionPaginator) Start() error {
	data := p.getPageData(*p.currentPage)
	err := p.session.InteractionRespond(p.interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
	if err != nil {
		return err
	}

	go p.beginButtonListener()
	return nil
}

func (p *InteractionPaginator) beginButtonListener() {
	curPage := *p.currentPage

	handler := func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		if ic.Type != discordgo.InteractionMessageComponent {
			return
		}

		switch ic.MessageComponentData().CustomID {
		case "first":
			curPage = 0
		case "prev":
			if curPage > 0 {
				curPage--
			}
		case "next":
			if curPage < p.totalPages-1 {
				curPage++
			}
		case "last":
			curPage = p.totalPages - 1
		default:
			return
		}

		data := p.getPageData(curPage)
		s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: data,
		})
	}

	p.session.AddHandler(handler)
	select {
	case <-time.After(p.timeout):
		p.Stop()
	case <-p.stopChan:
	}
}

func (p *Paginator) Stop() {
	close(p.stopChan)
}
