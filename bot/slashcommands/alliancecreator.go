package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

// Handles the submission of the modal for creating an alliance.
func handleAllianceCreatorModal(s *discordgo.Session, i *discordgo.Interaction) error {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	inputs := discordutil.GetModalInputs(i)

	ident := inputs["identifier"]
	if allianceStore.HasKey(strings.ToLower(ident)) {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nAn alliance with this identifier already exists.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	representativeUser, err := s.User(inputs["representative"])
	if err != nil {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nRepresentative ID does point to a valid Discord user.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	inputNations := strings.Split(strings.ReplaceAll(inputs["nations"], " ", ""), ",")
	if len(inputNations) < 2 {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nOnly one nation input specified, minimum two required.\n", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	//#region Check nations name inputs are valid and grab their UUIDs.
	validNations, missingNations := validateNations(nationStore, inputNations)
	if len(validNations) < 1 {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nNone of the input nation names were valid nations.\n", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	nationUUIDs := lo.SliceToMap(validNations, func(n oapi.NationInfo) (string, struct{}) {
		return n.UUID, struct{}{}
	})
	//#endregion

	//#region Validate parent alliance
	var parent *string
	parentInput := strings.ReplaceAll(inputs["parent"], " ", "")
	if parentInput != "" {
		pa, err := allianceStore.Get(strings.ToLower(parentInput))
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Parent alliance `%s` does not exist.", parentInput),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		parent = &pa.Identifier
	}
	//#endregion

	id, createdTs := generateAllianceID()
	cleanLabel := strings.TrimSpace(inputs["label"])

	alliance := database.Alliance{
		UUID:             id,
		Identifier:       ident,
		Label:            cleanLabel,
		RepresentativeID: &representativeUser.ID,
		OwnNations:       nationUUIDs,
		Parent:           parent,
		Type:             database.AllianceTypePact,
		UpdatedTimestamp: &createdTs,
	}

	allianceStore.Set(strings.ToLower(ident), alliance)

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	embed := embeds.NewAllianceEmbed(s, allianceStore, alliance, nil)
	content := "Successfully created alliance:"
	if len(missingNations) > 0 {
		embed.Color = discordutil.GOLD
		content = fmt.Sprintf(
			"Partially created alliance, the following nations were invalid:```%s```",
			strings.Join(missingNations, ", "),
		)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	return nil
}

func createAlliance(s *discordgo.Session, i *discordgo.Interaction) error {
	isEditor, _ := discordutil.HasRole(i.Member, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	// See handleAllianceCreatorModal() for submission handling,
	// where the actual creation and saving to the database occurs.
	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_creator",
		Title:    "Alliance Creator",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "identifier",
				Label:       "Query Identifier (3-16 chars)",
				Placeholder: "Enter a unique short name used to query this alliance.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   3,
				MaxLength:   16,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "label",
				Label:       "Alliance Name (4-64 chars)",
				Placeholder: "Enter this alliance's full name.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   64,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "representative",
				Label:       "Representative Discord ID",
				Placeholder: "Enter the Discord ID of the user representing this alliance.",
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
				CustomID:    "parent",
				Label:       "Parent Alliance",
				Placeholder: "(Optional) Enter the identifier of this alliance's parent alliance.",
				Required:    false,
				MinLength:   3,
				MaxLength:   16,
				Style:       discordgo.TextInputShort,
			}),
		},
	})
}

func disbandAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	isSrEditor, _ := discordutil.HasRole(i.Member, SR_EDITOR_ROLE)
	if !isSrEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Only senior editors can disband alliances.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	opt := cdata.GetOption("disband")
	ident := opt.GetOption("identifier").StringValue()

	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	a, _ := allianceStore.Find(func(a database.Alliance) bool {
		return strings.EqualFold(a.Identifier, ident)
	})
	if a == nil {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Cannot disband alliance `%s` as it does not exist.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	// Send disband notif and the alliance's json data to backup channel.
	sendAllianceBackup(s, i, a, "disbanded")

	allianceStore.Delete(strings.ToLower(a.Identifier))

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", a.Identifier, err)
	}

	_, err = discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: fmt.Sprintf("Successfully disbanded alliance `%s` aka `%s`.", a.Label, a.Identifier),
	})

	return err
}
