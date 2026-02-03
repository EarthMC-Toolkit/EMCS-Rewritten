package discordutil

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CustomNavigationButtonRow struct {
	First discordgo.Button
	Prev  discordgo.Button
	Next  discordgo.Button
	Last  discordgo.Button
}

const PAGINATOR_TIMEOUT = 3 * time.Minute

type Paginator struct {
	session     *discordgo.Session
	messageID   string
	authorID    string
	currentPage *int
	totalPages  int
	perPage     int
	timeout     time.Duration
}

// The total amount of pages that were determined to be required to show all existing data in this paginator.
//
// This number can never go below 1.
func (p *Paginator) TotalPages() int {
	if p.totalPages < 1 {
		log.Printf("WARNING | invalid value '%d' for totalPages. cannot be less than 1", p.totalPages)
		return 1
	}

	return p.totalPages
}

// Gets the start and end indexes for the items that should be on the current page.
//
// For example, when perPage is set to 10 and the totalItems given is 35, then every time the
// current page is incremented, the output each time would look like so: (0, 9), (10, 19), (20, 29), (30, 34).
func (p *Paginator) CurrentPageBounds(totalItems int) (int, int) {
	start := *p.currentPage * p.perPage
	return start, min(start+p.perPage, totalItems)
}

func (p *Paginator) NewNavigationButtonRow() *discordgo.ActionsRow {
	curPage := *p.currentPage

	return &discordgo.ActionsRow{
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

type InteractionPageFunc func(curPage int, data *discordgo.InteractionResponseData)

type InteractionPaginator struct {
	*Paginator
	interaction  *discordgo.Interaction
	cache        map[int]*discordgo.InteractionResponseData
	PageFunc     InteractionPageFunc
	customNavRow *discordgo.ActionsRow // Must be a set of buttons with ids "first", "prev", "next" and "last"
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
			session:     s,
			authorID:    author.ID,
			currentPage: &initPage,
			totalPages:  totalPages,
			perPage:     perPage,
			timeout:     PAGINATOR_TIMEOUT,
		},
	}
}

func (p *InteractionPaginator) WithCustomNavigationRow(row CustomNavigationButtonRow) *InteractionPaginator {
	row.First.CustomID = "first"
	row.Prev.CustomID = "prev"
	row.Next.CustomID = "next"
	row.Last.CustomID = "last"

	p.customNavRow = &discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			row.First,
			row.Prev,
			row.Next,
			row.Last,
		},
	}

	return p
}

func (p *InteractionPaginator) WithTimeout(t time.Duration) *InteractionPaginator {
	p.timeout = t
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

// TODO: Do we even need to store a pointer to data? We could just return data as value from PageFunc
func (p *InteractionPaginator) getPageData(page int) *discordgo.InteractionResponseData {
	data, ok := p.cache[page]
	if !ok {
		data = &discordgo.InteractionResponseData{}
		p.PageFunc(page, data)
		p.cache[page] = data
	}

	cpy := *data // Work with shallow copy of the cached page
	if p.totalPages > 1 {
		row := p.customNavRow
		if row == nil {
			row = p.NewNavigationButtonRow()
		}

		// Always make navigation row the first row.
		cpy.Components = append([]discordgo.MessageComponent{row}, cpy.Components...)
	}

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
		if author := GetInteractionAuthor(ic.Interaction); author.ID != p.authorID {
			return
		}

		switch ic.MessageComponentData().CustomID {
		case "first":
			*p.currentPage = 0
		case "prev":
			if *p.currentPage > 0 {
				*p.currentPage--
			}
		case "next":
			if *p.currentPage < p.totalPages-1 {
				*p.currentPage++
			}
		case "last":
			*p.currentPage = p.totalPages - 1
		default:
			return
		}

		DeferComponent(s, ic.Interaction)

		data := p.getPageData(*p.currentPage)
		EditReply(s, ic.Interaction, data)
	}

	removeHandler := p.session.AddHandler(btnHandler)
	go func() {
		time.Sleep(p.timeout)

		row := p.customNavRow
		if row == nil {
			row = p.NewNavigationButtonRow()
		}

		// Disable all buttons now paginator expired
		for i, comp := range row.Components {
			if btn, ok := comp.(discordgo.Button); ok {
				btn.Disabled = true
				row.Components[i] = btn
			}
		}

		// Edit message with new row of disabled btns
		_, _ = p.session.InteractionResponseEdit(p.interaction, &discordgo.WebhookEdit{
			Components: &[]discordgo.MessageComponent{row},
		})

		removeHandler()
	}()
}
