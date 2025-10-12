package discordutil

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const PAGINATOR_TIMEOUT = 5 * time.Minute

type Paginator struct {
	session     *discordgo.Session
	messageID   string
	authorID    string
	currentPage *int
	totalPages  int
	perPage     int
	timeout     time.Duration
}

func (p *Paginator) TotalPages() int {
	return p.totalPages
}

// Gets the start and end indexes for the items that should be on the current page. For example:
//
//	perPage = 50
//	totalItems = 100
//
// If currentPage is 0: output is (0, 50). If currentPage is 1: output is (51, 100).
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

// Creates a new InteractionPaginator which is a Paginator with extended functionality to work with InteractionResponseData.
// Default timeout is specified by PAGINATOR_TIMEOUT. To customise it, chain .WithTimeout() and pass a valid [time.Duration].
func NewInteractionPaginator(s *discordgo.Session, i *discordgo.Interaction, totalItems, perPage int) *InteractionPaginator {
	author := GetInteractionAuthor(i)

	initPage := 0
	totalPages := (totalItems + perPage - 1) / perPage // round to next largest int (ceil)

	return &InteractionPaginator{
		interaction: i,
		cache:       make(map[int]*discordgo.InteractionResponseData),
		Paginator: &Paginator{
			session: s,
			//channelID:   i.ChannelID,
			authorID:    author.ID,
			currentPage: &initPage,
			totalPages:  totalPages,
			perPage:     perPage,
			timeout:     PAGINATOR_TIMEOUT,
		},
	}
}

func (p *InteractionPaginator) WithTimeout(timeout time.Duration) *InteractionPaginator {
	p.timeout = timeout
	return p
}

func (p *InteractionPaginator) Start() error {
	data := p.getPageData(*p.currentPage)

	msg, err := EditOrSendReply(p.session, p.interaction, data)
	if err != nil {
		return err
	}

	if msg == nil {
		msg, err = p.session.InteractionResponse(p.interaction)
		if err != nil {
			return err
		}
	}

	p.messageID = msg.ID

	// TODO: This could overlap with the global button handler we setup in bot.Run()
	go p.startButtonListener()

	return nil
}

func (p *InteractionPaginator) getPageData(page int) *discordgo.InteractionResponseData {
	data, ok := p.cache[page]
	if !ok {
		data = &discordgo.InteractionResponseData{}
		p.PageFunc(page, data)
		p.cache[page] = data
	}

	cpy := *data
	return &cpy
}

func (p *InteractionPaginator) startButtonListener() {
	btnHandler := func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		if ic.Type != discordgo.InteractionMessageComponent {
			return
		}

		if ic.Message == nil || ic.Message.ID != p.messageID {
			return
		}

		author := GetInteractionAuthor(ic.Interaction)
		if author.ID != p.authorID {
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

	removeHandler := p.session.AddHandler(btnHandler)
	go func() {
		time.Sleep(p.timeout)
		removeHandler()
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

// Unlike a shallow copy or sharing the original pointer, a deep copy will prevent data of
// paginator instances from colliding and potentially showing/editing data of an irrelevent command.
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
