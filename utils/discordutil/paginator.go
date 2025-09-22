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
	totalPages  int
	perPage     int
	timeout     time.Duration
	stopChan    chan struct{}
}

func (p *Paginator) TotalPages() int {
	return p.totalPages
}

func (p *Paginator) CurrentPageBounds(totalItems int) (int, int) {
	start := *p.currentPage * p.perPage
	return start, min(start+p.perPage, totalItems)
}

type InteractionPageFunc func(curPage int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	interaction *discordgo.Interaction
	cache       map[int]*discordgo.InteractionResponseData
	PageFunc    InteractionPageFunc
}

func NewInteractionPaginator(s *discordgo.Session, i *discordgo.Interaction, totalItems, perPage int) *InteractionPaginator {
	author := UserFromInteraction(i)

	initPage := 0
	totalPages := (totalItems + perPage - 1) / perPage // round to next largest int (ceil)

	return &InteractionPaginator{
		interaction: i,
		cache:       make(map[int]*discordgo.InteractionResponseData),
		Paginator: &Paginator{
			session:     s,
			channelID:   i.ChannelID,
			userID:      author.ID,
			currentPage: &initPage,
			totalPages:  totalPages,
			perPage:     perPage,
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
				Disabled: curPage == p.totalPages-1,
			},
			discordgo.Button{
				Label:    ">>",
				CustomID: "last",
				Style:    discordgo.PrimaryButton,
				Disabled: curPage == p.totalPages-1,
			},
		},
	}
}

func (p *InteractionPaginator) getPageData(page int) *discordgo.InteractionResponseData {
	// Use the cached data for this page if available.
	if data, ok := p.cache[page]; ok {
		return data
	}

	data := &discordgo.InteractionResponseData{}
	p.PageFunc(page, data)
	p.cache[page] = data

	return data
}

func (p *InteractionPaginator) Start() (err error) {
	data := p.getPageData(*p.currentPage)

	_, err = EditOrSendReply(p.session, p.interaction, data)
	if err == nil {
		go p.beginButtonListener()
	}

	return
}

func (p *InteractionPaginator) beginButtonListener() {
	handler := func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		if ic.Type != discordgo.InteractionMessageComponent {
			return
		}

		switch ic.MessageComponentData().CustomID {
		case "first":
			*p.currentPage = 0
		case "prev":
			*p.currentPage--
		case "next":
			*p.currentPage++
		case "last":
			*p.currentPage = p.totalPages - 1
		default:
			return
		}

		// clamp to valid range
		if *p.currentPage < 0 {
			*p.currentPage = 0
		}
		if *p.currentPage >= p.totalPages {
			*p.currentPage = p.totalPages - 1
		}

		data := p.getPageData(*p.currentPage)
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
