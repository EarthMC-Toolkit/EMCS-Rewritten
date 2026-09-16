package slashcommands

import (
	"cmp"
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/utils/discordutil"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

const DEFAULT_ARTICLE_COUNT = uint8(10)

type NewsCommand struct{}

func (cmd NewsCommand) Name() string { return "news" }
func (cmd NewsCommand) Description() string {
	return "Retrieve news articles provided by the current news provider."
}

func (cmd NewsCommand) Options() []AppCommandOpt {
	return []AppCommandOpt{
		discordutil.SubcommandOption("latest", "Shows the latest articles.",
			discordutil.IntegerOption("count", "The number of news entries to show (max 20).", 1, 20, false),
		),
		discordutil.SubcommandOption("changelogs", "Shows a list of all reported server changelogs."),
		discordutil.SubcommandOption("search", "Shows all news articles relating to the specified term.",
			discordutil.RequiredStringOption("term", "The text to match news headlines by.", 2, 60),
		),
	}
}

func (cmd NewsCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	newsStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NEWS_STORE)
	if err != nil {
		return err
	}

	articles := newsStore.Values()

	cdata := i.ApplicationCommandData()
	opt := cdata.GetOption("latest")
	if opt != nil {
		return executeLatestNews(s, i.Interaction, opt, articles)
	}
	opt = cdata.GetOption("changelogs")
	if opt != nil {
		return executeChangelogNews(s, i.Interaction, articles)
	}
	opt = cdata.GetOption("search")
	if opt != nil {
		return executeSearchNews(s, i.Interaction, opt, articles)
	}

	discordutil.ReplyWithError(s, i.Interaction, errors.New("Subcommand not implemented yet."))
	return err
}

func executeLatestNews(
	s *discordgo.Session, i *discordgo.Interaction,
	opt *discordgo.ApplicationCommandInteractionDataOption,
	articles []database.NewsEntry,
) error {
	count := DEFAULT_ARTICLE_COUNT
	opt = opt.GetOption("count")
	if opt != nil {
		count = uint8(opt.IntValue())
	}

	filterSortArticles(articles, "")
	return sendPaginator(s, i, articles, int(count), "News Articles | Latest")
}

func executeChangelogNews(
	s *discordgo.Session, i *discordgo.Interaction,
	articles []database.NewsEntry,
) error {
	filterSortArticles(articles, "changelog")
	return sendPaginator(s, i, articles, 10, "News Articles | Changelogs")
}

func executeSearchNews(
	s *discordgo.Session, i *discordgo.Interaction,
	opt *discordgo.ApplicationCommandInteractionDataOption,
	articles []database.NewsEntry,
) error {
	term := strings.ToLower(opt.GetOption("term").StringValue())
	filterSortArticles(articles, term)
	return sendPaginator(s, i, articles, 10, fmt.Sprintf("News Articles | Search by term: `%s`", term))
}

// Filters articles by the search term and sorts them by timestamp in descending order.
// If the term is empty, filtering is skipped and only sorting is performed.
func filterSortArticles(articles []database.NewsEntry, filterTerm string) {
	if filterTerm != "" {
		articles = lo.Filter(articles, func(e database.NewsEntry, _ int) bool {
			return strings.Contains(strings.ToLower(e.Headline), filterTerm)
		})
	}
	slices.SortFunc(articles, func(a, b database.NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})
}

// Sends a paginator for the given articles, with the specified title and description.
func sendPaginator(
	s *discordgo.Session, i *discordgo.Interaction,
	articles []database.NewsEntry, perPage int, title string,
) error {
	paginator := discordutil.NewInteractionPaginator(s, i, len(articles), perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(len(articles))
		pageArticles := articles[start:end]
		pageArticlesCount := uint8(len(pageArticles))

		desc, _ := shared.BuildNewsString(pageArticles, pageArticlesCount, discordutil.EMBED_DESCRIPTION_LIMIT)
		pageTitle := fmt.Sprintf("[%d] %s | Page %d/%d", len(articles), title, curPage+1, paginator.TotalPages())

		embed := discordutil.NewEmbedBuilder(&discordutil.AQUA, &pageTitle, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}
