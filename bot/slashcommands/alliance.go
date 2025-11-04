package slashcommands

import (
	"emcsrw/bot/store"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const EDITOR_ROLE = "966359842417705020"
const SR_EDITOR_ROLE = "1143253762039873646"

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

		return queryAlliance(s, i.Interaction, cmdData)
	}
	if create := cmdData.GetOption("create"); create != nil {
		return createAlliance(s, i.Interaction, cmdData)
	}

	return nil
}

func queryAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
	ident := data.GetOption("query").GetOption("identifier").StringValue()

	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	alliance, err := allianceStore.GetKey(strings.ToLower(ident))
	if err != nil {
		fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	_, err = discordutil.FollowUpEmbeds(s, i, shared.NewAllianceEmbed(s, alliance))
	return err
}

func createAlliance(s *discordgo.Session, i *discordgo.Interaction, _ discordgo.ApplicationCommandInteractionData) error {
	isEditor, _ := discordutil.HasRole(i, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

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
				Placeholder: "Enter the alliance's full name...",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "representative",
				Label:       "Alliance Representative (3-16 chars)",
				Placeholder: "Enter the Discord ID of the representative...",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   17,
				MaxLength:   19,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "nations",
				Label:       "Own Nations",
				Placeholder: "Enter a comma-seperated list of nations in THIS alliance only.",
				Required:    true,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "parents",
				Label:       "Parent Alliance(s)",
				Placeholder: "(Optional) Enter a comma-seperated list of alliance *identifiers* which this alliance is a child of.",
				Required:    false,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "leaders",
				Label:       "(Optional) Leaders(s)",
				Placeholder: "Enter a comma-seperated list of leader IGNs. Defaults to 'None'.",
				Required:    false,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "discord",
				Label:       "(Optional) Discord Server Code",
				Placeholder: "Enter a permanent invite code to the alliance's discord. Ex: 'aQCpGDANX' or 'earthmcnet'",
				Required:    false,
				MinLength:   8,
				Style:       discordgo.TextInputShort,
			}),
		},
	})
	if err != nil {
		return err
	}

	// Saving the alliance to DB is handled in the modal handler.
	// See handleAllianceCreatorModal()

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

// 	db := database.GetMapDB(common.ACTIVE_MAP)
// 	err := database.PutAlliance(db, createdAlliance)
// 	if err != nil {
// 		fmt.Printf("failed to put alliance %s into db:\n%v", ident, err)

// 		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not create alliance `%s`! Check the console.", ident))
// 		return err
// 	}

// 	_, err = discordutil.FollowUpContent(s, i, fmt.Sprintf("Successfully created alliance `%s (%s)`", label, ident))
// 	return err
// }
