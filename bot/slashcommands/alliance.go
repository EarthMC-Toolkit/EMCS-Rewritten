package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/utils/discordutil"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type AllianceCommand struct{}

func (cmd AllianceCommand) Name() string { return "alliance" }
func (cmd AllianceCommand) Description() string {
	return "Look up and alliance or request one be created/edited."
}

func (cmd AllianceCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about an alliance (meganation or pact between nations).",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance.",
			// Options: AppCommandOpts{
			// 	discordutil.RequiredStringOption("identifier", "The short unique name used to query the alliance.", 3, 16),
			// 	discordutil.RequiredStringOption("label", "The full name for display purposes.", 4, 36),
			// },
		},
	}
}

func (cmd AllianceCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cmdData := i.ApplicationCommandData()

	if lookup := cmdData.GetOption("query"); lookup != nil {
		err := discordutil.DeferReply(s, i.Interaction)
		if err != nil {
			return err
		}

		return QueryAlliance(s, i.Interaction, cmdData)
	}

	// if create := cmdData.GetOption("create"); create != nil {
	// 	return CreateAlliance(s, i.Interaction, cmdData)
	// }

	return nil
}

func QueryAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
	opt := data.GetOption("query")
	ident := opt.GetOption("identifier").StringValue()

	// Try find alliance in DB
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	alliance, err := database.GetAllianceByIdentifier(db, ident)
	if err != nil {
		fmt.Printf("failed to get alliance '%s' from db: %v", ident, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	_, err = discordutil.FollowUpEmbeds(s, i, common.NewAllianceEmbed(s, alliance))
	return err
}

func CreateAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
	err := discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_creator_modal",
		Title:    "Alliance Creator",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "identifier",
				Label:       "Query Identifier (3-16 chars)",
				Placeholder: "Enter a unique short name used to query the alliance...",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   3,
				MaxLength:   16,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "label",
				Label:       "Alliance Name (4-36 chars)",
				Placeholder: "Enter the full alliance name...",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "nations",
				Label:       "Nations",
				Placeholder: "Nation1, Nation2, Nation3...",
				Required:    true,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// func CreateAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
// 	opt := data.GetOption("create")

// 	ident := opt.GetOption("identifier").StringValue()
// 	label := opt.GetOption("label").StringValue()

// 	createdAlliance := &database.Alliance{
// 		ID:         generateAllianceID(),
// 		Identifier: ident,
// 		Label:      label,
// 	}

// 	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
// 	err := database.PutAlliance(db, createdAlliance)
// 	if err != nil {
// 		fmt.Printf("failed to put alliance %s into db:\n%v", ident, err)

// 		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not create alliance `%s`! Check the console.", ident))
// 		return err
// 	}

// 	_, err = discordutil.FollowUpContent(s, i, fmt.Sprintf("Successfully created alliance `%s (%s)`", label, ident))
// 	return err
// }
