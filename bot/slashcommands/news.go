package slashcommands

import (
	"cmp"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
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

func (cmd NewsCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "latest",
			Description: "Shows the latest articles.",
			Options: AppCommandOpts{
				discordutil.IntegerOption("count", "The number of news entries to show (max 20).", 1, 20, false),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "changelogs",
			Description: "Shows a list of all reported server changelogs.",
		},
		{
			// TODO: Maybe just use a 'term' option for searching instead of a nation-specific subcommand?
			// 		 Would be more flexible (alliances, towns etc) and future-proof.
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "search",
			Description: "Shows all news articles relating to the specified term.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("term", "The text to match news headlines by.", 2, 60),
			},
		},
		// {
		// 	Type:        discordgo.ApplicationCommandOptionSubCommand,
		// 	Name:        "alliance",
		// 	Description: "Shows all news articles relating to the the specified alliance.",
		// 	Options: AppCommandOpts{
		// 		discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
		// 	},
		// },
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

	slices.SortFunc(articles, func(a, b database.NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})

	desc, amt := shared.BuildNewsString(articles, count, discordutil.EMBED_DESCRIPTION_LIMIT)
	title := fmt.Sprintf("[%d] News Articles | Latest", amt)

	embed := discordutil.NewEmbedBuilder(&discordutil.AQUA, &title, &desc, nil)
	_, err := discordutil.FollowupEmbeds(s, i, embed.Build())
	return err
}

func executeChangelogNews(
	s *discordgo.Session, i *discordgo.Interaction,
	articles []database.NewsEntry,
) error {
	articles = lo.Filter(articles, func(e database.NewsEntry, _ int) bool {
		return strings.Contains(strings.ToLower(e.Headline), "changelog")
	})
	slices.SortFunc(articles, func(a, b database.NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})

	desc, amt := shared.BuildNewsString(articles, 10, discordutil.EMBED_DESCRIPTION_LIMIT)
	title := fmt.Sprintf("[%d] News Articles | Changelogs", amt)

	embed := discordutil.NewEmbedBuilder(&discordutil.AQUA, &title, &desc, nil)
	_, err := discordutil.FollowupEmbeds(s, i, embed.Build())
	return err
}

func executeSearchNews(
	s *discordgo.Session, i *discordgo.Interaction,
	opt *discordgo.ApplicationCommandInteractionDataOption,
	articles []database.NewsEntry,
) error {
	term := strings.ToLower(opt.GetOption("term").StringValue())

	articles = lo.Filter(articles, func(e database.NewsEntry, _ int) bool {
		return strings.Contains(strings.ToLower(e.Headline), term)
	})
	slices.SortFunc(articles, func(a, b database.NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})

	desc, amt := shared.BuildNewsString(articles, 20, discordutil.EMBED_DESCRIPTION_LIMIT)
	title := fmt.Sprintf("[%d] News Articles | Search by term: `%s`", amt, term)

	embed := discordutil.NewEmbedBuilder(&discordutil.AQUA, &title, &desc, nil)
	_, err := discordutil.FollowupEmbeds(s, i, embed.Build())
	return err
}
