package events

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/store"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"path"
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
			//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

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
	inputs := modalInputsToMap(i)

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
	inputs := modalInputsToMap(i)

	image := inputs["image"]
	if image != "" {
		parsedUrl, err := validateAllianceImage(image)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Input for field **Image/Flag URL** could not be parsed correctly. Reason:\n```%s```", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		// Ensure we always use the original cdn link.
		image = parsedUrl
	}

	var discordCode string
	if alliance.Optional.DiscordCode != nil {
		discordCode = *alliance.Optional.DiscordCode
	}

	if input := inputs["discord"]; input != "" {
		code, err := discordutil.ExtractInviteCode(inputs["discord"])
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: "Input for field **Discord Invite** could not be parsed correctly. Provide a link or code.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		_, err = discordutil.ValidateInviteCode(code, s)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Input for field **Discord Invite** was parsed correctly but could not be used. Reason:\n```%s```", err.Error()),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		discordCode = code
	}

	//leaders := strings.Split(strings.ReplaceAll(inputs["leaders"], " ", ""), ",")
	// validate leaders exist

	//colours := inputs["colours"]
	// validate hex

	allianceType := store.NewAllianceType(inputs["type"]) // invalid input will default to pact

	// Update alliance fields after all validation/transformations complete.
	alliance.Type = allianceType
	alliance.Optional.DiscordCode = &discordCode
	alliance.Optional.ImageURL = &image

	// Update alliance in store
	allianceStore.SetKey(strings.ToLower(alliance.Identifier), *alliance)

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err := allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. WriteSnapshot failed", alliance.Identifier)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully edited alliance:",
		Embeds: []*discordgo.MessageEmbed{
			shared.NewAllianceEmbed(s, alliance),
		},
	})

	return nil
}

// Handles the submission of the modal for creating an alliance.
func handleAllianceCreatorModal(s *discordgo.Session, i *discordgo.Interaction) error {
	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	inputs := modalInputsToMap(i)

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

func generateAllianceID() uint64 {
	createdTs := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))        // Safe to cast to uint since Intn returns 0-n anyway.
	return (createdTs << 16) | suffix
}

func modalInputsToMap(i *discordgo.Interaction) map[string]string {
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

func validateAllianceImage(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.New("not even close to being a URL")
	}

	if u.Scheme != "https" {
		return "", errors.New("must use https")
	}

	switch u.Host {
	case "cdn.discordapp.com":
	case "media.discordapp.net":
		u.Host = "cdn.discordapp.com"
		u.RawQuery = "" // remove resize/compression params
	default:
		return "", errors.New("url must point to Discord's CDN or media domain")
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 5 || parts[1] != "attachments" {
		return "", errors.New("invalid discord image attachment path")
	}

	if parts[2] != shared.FLAGS_CHANNEL_ID {
		return "", fmt.Errorf("wrong channel. use image from flags channel")
	}

	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
	default:
		return "", errors.New("url does not point to an image type")
	}

	return u.String(), nil
}
