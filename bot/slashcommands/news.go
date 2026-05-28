package slashcommands

import (
	"emcsrw/utils/discordutil"
	"errors"

	"github.com/bwmarrin/discordgo"
)

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
	// newsStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NEWS_STORE)
	// if err != nil {
	// 	return err
	// }

	// articles := newsStore.Values()

	discordutil.ReplyWithError(s, i.Interaction, errors.New("Command not implemented yet."))
	return nil
}
