package events

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/store"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"math/rand"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

func OnInteractionCreateModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateModalSubmit recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionModalSubmit {
		return
	}

	data := i.ModalSubmitData()
	if data.CustomID == "alliance_creator" {
		err := handleAllianceCreatorModal(s, i.Interaction)
		if err != nil {
			discordutil.ReplyWithError(s, i.Interaction, err)
			return
		}
	}

	if strings.HasPrefix(data.CustomID, "alliance_editor") {
		allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
		if err != nil {
			return
		}

		ident := strings.Split(data.CustomID, "@")[1]
		alliance, err := allianceStore.GetKey(strings.ToLower(ident))
		if err != nil {
			fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

			discordutil.FollowUpContentEphemeral(s, i.Interaction, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
			return
		}

		if strings.HasPrefix(data.CustomID, "alliance_editor_functional") {
			err := handleAllianceEditorModalFunctional(s, i.Interaction, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i.Interaction, err)
				return
			}
		}

		if strings.Contains(data.CustomID, "alliance_editor_optional") {
			err := handleAllianceEditorModalOptional(s, i.Interaction, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i.Interaction, err)
				return
			}
		}
	}
}

func handleAllianceEditorModalFunctional(
	s *discordgo.Session, i *discordgo.Interaction,
	alliance *store.Alliance, allianceStore *store.Store[store.Alliance],
) error {
	inputs := ModalInputsToMap(i)

	oldIdent := alliance.Identifier

	ident := defaultIfEmpty(inputs["identifier"], oldIdent)
	label := defaultIfEmpty(inputs["label"], alliance.Label)
	parents := defaultIfEmpty(inputs["parents"], strings.Join(alliance.Parents, ", "))

	representative := inputs["representative"]
	representativeID := alliance.RepresentativeID
	if representative != "" {
		// Validate representative is existing Discord user.
		representativeUser, err := s.User(representative)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: "Representative ID does not point to a valid Discord user.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		representativeID = &representativeUser.ID
	}

	ownNations := alliance.OwnNations
	if inputs["nations"] != "" {
		nationsInput := strings.Split(strings.ReplaceAll(inputs["nations"], " ", ""), ",")

		//#region Get UUIDs from nation names
		nameSet := make(store.ExistenceMap, len(nationsInput))
		for _, name := range nationsInput {
			nameSet[strings.ToLower(name)] = struct{}{}
		}

		nationStore, _ := store.GetStoreForMap[oapi.NationInfo](shared.ACTIVE_MAP, "nations")
		nations := nationStore.FindMany(func(n oapi.NationInfo) bool {
			_, ok := nameSet[strings.ToLower(n.Name)]
			return ok
		})

		ownNations = lo.Map(nations, func(n oapi.NationInfo, _ int) string {
			return n.UUID
		})
		//#endregion
	}

	// Update after all validation/transformations complete.
	alliance.Identifier = ident
	alliance.Label = label
	alliance.RepresentativeID = representativeID
	alliance.OwnNations = ownNations
	alliance.Parents = strings.Split(strings.ReplaceAll(parents, " ", ""), ",")

	// Update store
	allianceStore.SetKey(strings.ToLower(ident), *alliance)
	if !strings.EqualFold(oldIdent, alliance.Identifier) {
		allianceStore.DeleteKey(strings.ToLower(oldIdent)) // remove old key if identifier changed
	}

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err := allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. WriteSnapshot failed", ident)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully edited alliance:",
		Embeds: []*discordgo.MessageEmbed{
			shared.NewAllianceEmbed(s, alliance),
		},
	})

	return nil
}

func handleAllianceEditorModalOptional(
	s *discordgo.Session, i *discordgo.Interaction,
	alliance *store.Alliance, allianceStore *store.Store[store.Alliance],
) error {
	//inputs := ModalInputsToMap(i)
	return nil
}

// Handles the submission of the modal for creating an alliance.
func handleAllianceCreatorModal(s *discordgo.Session, i *discordgo.Interaction) error {
	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	inputs := ModalInputsToMap(i)

	ident := inputs["identifier"]
	if allianceStore.HasKey(strings.ToLower(ident)) {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nAn alliance with this identifier already exists.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	label := inputs["label"]
	nationsInput := strings.Split(strings.ReplaceAll(inputs["nations"], " ", ""), ",")

	representativeUser, err := s.User(inputs["representative"])
	if err != nil {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Representative ID does point to a valid Discord user.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	//#region Get UUIDs from nation names
	nameSet := make(store.ExistenceMap, len(nationsInput))
	for _, name := range nationsInput {
		nameSet[strings.ToLower(name)] = struct{}{}
	}

	nationStore, _ := store.GetStoreForMap[oapi.NationInfo](shared.ACTIVE_MAP, "nations")
	nations := nationStore.FindMany(func(n oapi.NationInfo) bool {
		_, ok := nameSet[strings.ToLower(n.Name)]
		return ok
	})

	nationUUIDs := lo.Map(nations, func(n oapi.NationInfo, _ int) string {
		return n.UUID
	})
	//#endregion

	alliance := store.Alliance{
		UUID:             generateAllianceID(),
		Identifier:       ident,
		Label:            label,
		RepresentativeID: &representativeUser.ID,
		OwnNations:       nationUUIDs,
	}

	allianceStore.SetKey(strings.ToLower(ident), alliance)

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully created alliance:",
		Embeds: []*discordgo.MessageEmbed{
			shared.NewAllianceEmbed(s, &alliance),
		},
	})

	//fmt.Print(utils.Prettify(alliance))
	return nil
}

func ModalInputsToMap(i *discordgo.Interaction) map[string]string {
	inputs := make(map[string]string)
	for _, row := range i.ModalSubmitData().Components {
		actionRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue // Must not be an action row, we don't care.
		}

		// Gather values of all text input components in this row.
		for _, comp := range actionRow.Components {
			if input, ok := comp.(*discordgo.TextInput); ok {
				inputs[input.CustomID] = input.Value
			}
		}
	}

	return inputs
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func generateAllianceID() uint64 {
	createdTs := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))        // Safe to cast to uint since Intn returns 0-n anyway.
	return (createdTs << 16) | suffix
}
