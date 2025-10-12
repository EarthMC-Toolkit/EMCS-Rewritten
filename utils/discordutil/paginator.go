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

// Gets the start and end indexes for the items that should be on the current page. For example:
//
//	perPage = 50
//	totalItems = 100
//
// If currentPage is 0: output is (0, 50). If currentPage is 1: output is (50, 100).
func (p *Paginator) CurrentPageBounds(totalItems int) (int, int) {
	start := *p.currentPage * p.perPage
	return start, min(start+p.perPage, totalItems)
}

func (p *Paginator) Stop() {
	close(p.stopChan)
}

type InteractionPageFunc func(curPage int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	interaction *discordgo.Interaction
	cache       map[int]*discordgo.InteractionResponseData
	PageFunc    InteractionPageFunc
}

func NewInteractionPaginator(s *discordgo.Session, i *discordgo.Interaction, totalItems, perPage int) *InteractionPaginator {
	author := GetInteractionAuthor(i)

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

func (p *InteractionPaginator) Start() error {
	data := p.getPageData(*p.currentPage)

	_, err := EditOrSendReply(p.session, p.interaction, data)
	if err == nil {
		go p.beginButtonListener()
	}

	return err
}

func (p *InteractionPaginator) getPageData(page int) *discordgo.InteractionResponseData {
	// Use the cached data for this page if available.
	if data, ok := p.cache[page]; ok {
		return data
	}

	data := &discordgo.InteractionResponseData{}
	p.PageFunc(page, data)

	// We use a deep copy here instead of sharing the original pointer. Shallow copy likely won't be enough.
	// This prevents data of paginator instances from colliding and potentially showing data from an irrelevent command.
	p.cache[page] = data

	return data
}

func (p *InteractionPaginator) beginButtonListener() {
	handler := func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		if ic.Type != discordgo.InteractionMessageComponent {
			return
		}

		// Only handle interactions for this paginator’s message
		if ic.Message == nil || ic.Message.ID != p.interaction.Message.ID {
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
	go func() {
		select {
		case <-time.After(p.timeout):
			p.Stop()
		case <-p.stopChan:
			return
		}
	}()
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

// func deepCopyInteractionData(data *discordgo.InteractionResponseData) *discordgo.InteractionResponseData {
// 	b, err := json.Marshal(data)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var cpy discordgo.InteractionResponseData
// 	if err := json.Unmarshal(b, &cpy); err != nil {
// 		panic(err)
// 	}

// 	return &cpy
// }
