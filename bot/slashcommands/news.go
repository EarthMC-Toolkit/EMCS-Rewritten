package slashcommands

import (
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils/discordutil"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
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
			// TODO: Maybe just use a 'term' option for searching instead of a nation-specific subcommand?
			// 		 Would be more flexible (alliances, towns etc) and future-proof.
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "nation",
			Description: "Shows all news articles relating to the the specified nation.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the nation to query.", 2, 40, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "changelogs",
			Description: "Shows a list of all reported server changelogs.",
		},
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

	title := fmt.Sprintf("News Articles | %d Most Recent", count)
	desc, _ := embeds.BuildNewsString(articles, count, discordutil.EMBED_DESCRIPTION_LIMIT)

	embed := discordutil.NewEmbed(&discordutil.AQUA, &title, &desc, nil)
	_, err := discordutil.FollowupEmbeds(s, i, embed)
	return err
}
