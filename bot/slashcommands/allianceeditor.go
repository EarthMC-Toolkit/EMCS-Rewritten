package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/sets"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

type MultiUpdateResult struct {
	AddedTo          map[string][]string
	RemovedFrom      map[string][]string
	InvalidAlliances sets.Set[string]
	InvalidNations   sets.Set[string]
	AlreadyPuppets   map[string][]string
	ChangesWritten   bool
}

func editAlliance(s *discordgo.Session, i *discordgo.Interaction) error {
	isEditor, _ := discordutil.HasRole(i.Member, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	// get the sub cmd currently being used
	subCmd := discordutil.GetActiveSubCommand(i.ApplicationCommandData())
	if subCmd == nil {
		return fmt.Errorf("no valid option found for editAlliance")
	}
	if subCmd.Name == "multi" {
		return openEditorModalMultiUpdate(s, i)
	}

	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	ident := subCmd.GetOption("identifier").StringValue()
	alliance, err := allianceStore.Get(strings.ToLower(ident))
	if err != nil {
		//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	switch subCmd.Name {
	// /alliance edit
	case "functional":
		return openEditorModalFunctional(s, i, alliance)
	case "optional":
		return openEditorModalOptional(s, i, alliance)
	// /alliance update
	case "nations":
		return openEditorModalNationsUpdate(s, i, alliance)
	case "leaders":
		return openEditorModalLeadersUpdate(s, i, alliance)
	}

	return nil // unreachable
}

func openEditorModalFunctional(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	nationStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	nations := nationStore.GetFromSet(alliance.OwnNations)
	nationNames := lo.Map(nations, func(n oapi.NationInfo, _ int) string { return n.Name })

	nationsPlaceholder := "Too many nations to display, run /alliance query to see the full list."
	nationsStr := strings.Join(nationNames, ", ")
	if len(nationsStr) < 100 {
		nationsPlaceholder = nationsStr
	}

	parentPlaceholder := ""
	if alliance.Parent != nil {
		parentPlaceholder = *alliance.Parent
	}

	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_functional@" + alliance.Identifier,
		Title:    "Alliance Editor - Functional Fields",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "identifier",
				Label:       "Query Identifier (3-16 chars)",
				Placeholder: alliance.Identifier,
				Style:       discordgo.TextInputShort,
				MinLength:   3,
				MaxLength:   16,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "label",
				Label:       "Alliance Name (4-64 chars)",
				Placeholder: alliance.Label,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   64,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "representative",
				Label:       "Representative Discord ID",
				Placeholder: *alliance.RepresentativeID,
				Style:       discordgo.TextInputShort,
				MinLength:   17,
				MaxLength:   19,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "nations",
				Label:       "Own Nations",
				Placeholder: nationsPlaceholder,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "parent",
				Label:       "Parent Alliance",
				Placeholder: parentPlaceholder,
				MinLength:   3,
				MaxLength:   16,
				Style:       discordgo.TextInputShort,
			}),
		},
	})
}

func openEditorModalOptional(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	entitiesStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ENTITIES_STORE)
	if err != nil {
		return err
	}

	reslist, _ := entitiesStore.Get("residentlist")
	townlesslist, _ := entitiesStore.Get("townlesslist")

	discordPlaceholder := "Enter an invite link or code to the alliance's Discord."
	if alliance.Optional.DiscordCode != nil {
		discordPlaceholder = fmt.Sprintf("https://discord.gg/%s", *alliance.Optional.DiscordCode)
	}

	imagePlaceholder := "Enter the URL of the alliance's image/flag from the flags channel."
	if alliance.Optional.ImageURL != nil {
		imagePlaceholder = *alliance.Optional.ImageURL
		if len(imagePlaceholder) >= 100 {
			imagePlaceholder = "Flag image URL too long to display!"
		}
	}

	leaderPlaceholder := "Enter the Minecraft IGNs of the alliance leaders, comma-separated."
	if alliance.Optional.Leaders != nil {
		leaderNames := alliance.GetLeaderNames(reslist, townlesslist)
		leaderPlaceholder = strings.Join(leaderNames, ", ")
	}

	coloursPlaceholder := "Enter HEX colour(s) seperated by a space. Fill first, Outline second."

	colours := alliance.Optional.Colours
	if colours != nil && colours.Fill != nil {
		fill := strings.TrimPrefix(*colours.Fill, "#")
		coloursPlaceholder = "#" + fill

		if colours.Outline != nil {
			coloursPlaceholder += " #" + strings.TrimPrefix(*colours.Outline, "#")
		} else {
			coloursPlaceholder += " #" + fill
		}
	}

	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_optional@" + alliance.Identifier,
		Title:    "Alliance Editor - Optional Fields",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "type",
				Label:       "Alliance Type (mega/org/pact)",
				Placeholder: string(alliance.Type),
				MinLength:   3,
				MaxLength:   4,
				Style:       discordgo.TextInputShort,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "discord",
				Label:       "Permanent Discord Invite",
				Placeholder: discordPlaceholder,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   40,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "image",
				Label:       "Image/Flag URL",
				Placeholder: imagePlaceholder,
				Style:       discordgo.TextInputShort,
				MinLength:   20,
				MaxLength:   500,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "colours",
				Label:       "Colours (Used by bot & extension)",
				Placeholder: coloursPlaceholder,
				MinLength:   4,
				Style:       discordgo.TextInputShort,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "leaders",
				Label:       "Leader IGNs (comma-separated)",
				Placeholder: leaderPlaceholder,
				Style:       discordgo.TextInputParagraph,
				MinLength:   3,
				MaxLength:   320, // Minecraft max name length is 16. Should suffice for many leaders.
			}),
		},
	})
}

func openEditorModalMultiUpdate(s *discordgo.Session, i *discordgo.Interaction) error {
	//requireSelectMenu := false
	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_multi",
		Title:    "Alliance Editor - Multi",
		Flags:    discordgo.MessageFlagsIsComponentsV2,
		Components: []discordgo.MessageComponent{
			// discordutil.Label("Add method", "Add nations directly or to puppets", discordgo.SelectMenu{
			// 	CustomID: "alliances-add-method",
			// 	MenuType: discordgo.StringSelectMenu,
			// 	Required: &requireSelectMenu,
			// 	Options: []discordgo.SelectMenuOption{
			// 		{Label: "Direct/Parent only", Value: "direct", Default: true},
			// 		{Label: "Add to puppets (if applicable)", Value: "puppets"},
			// 	},
			// }),
			discordutil.Label("Nations to add", "Please use a comma-seperated list", discordgo.TextInput{
				CustomID:    "nations-add",
				Placeholder: "Enter list of nation names...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
			discordutil.Label("Alliances to add the nations to", "Please use a comma-seperated list", discordgo.TextInput{
				CustomID:    "alliances-add",
				Placeholder: "Enter list of alliance identifiers..",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
			discordutil.Label("Nations to remove", "Please use a comma-seperated list", discordgo.TextInput{
				CustomID:    "nations-remove",
				Placeholder: "Enter list of nation names...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
			discordutil.Label("Alliances to remove the nations from", "Please use a comma-seperated list", discordgo.TextInput{
				CustomID:    "alliances-remove",
				Placeholder: "Enter list of alliance identifiers..",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
		},
	})
}

func openEditorModalNationsUpdate(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_nations@" + alliance.Identifier,
		Title:    "Alliance Editor - Nations Field",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "add",
				Label:       "Nations to Add (comma-seperated)",
				Placeholder: "Enter list of nation names...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "remove",
				Label:       "Nations to Remove (comma-seperated)",
				Placeholder: "Enter list of nation names...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   2,
			}),
		},
	})
}

func openEditorModalLeadersUpdate(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_leaders@" + alliance.Identifier,
		Title:    "Alliance Editor - Leaders Field",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "add",
				Label:       "Leaders to Add (comma-seperated)",
				Placeholder: "Enter list of IGNs...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   3,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "remove",
				Label:       "Leaders to Remove (comma-seperated)",
				Placeholder: "Enter list of IGNs...",
				Style:       discordgo.TextInputParagraph,
				MinLength:   3,
			}),
		},
	})
}

func handleAllianceEditorModalMultiUpdate(
	s *discordgo.Session, i *discordgo.Interaction,
	allianceStore *store.Store[database.Alliance],
) error {
	inputs := discordutil.GetModalInputs(i)
	if len(inputs) == 0 {
		if _, err := discordutil.FollowupContentEphemeral(s, i, "No inputs entered in the modal. Are you high?"); err != nil {
			return err
		}
	}

	addInput := make(map[string][]string)
	if v := strings.TrimSpace(inputs["alliances-add"]); v != "" {
		if alliances, err := utils.ParseFieldsStr(v, ','); err == nil {
			nations, _ := utils.ParseFieldsStr(strings.TrimSpace(inputs["nations-add"]), ',')
			for _, a := range alliances {
				addInput[a] = nations
			}
		}
	}

	removeInput := make(map[string][]string)
	if v := strings.TrimSpace(inputs["alliances-remove"]); v != "" {
		if alliances, err := utils.ParseFieldsStr(v, ','); err == nil {
			nations, _ := utils.ParseFieldsStr(strings.TrimSpace(inputs["nations-remove"]), ',')
			for _, a := range alliances {
				removeInput[a] = nations
			}
		}
	}

	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	result := MultiUpdateAlliances(allianceStore, nationStore, addInput, removeInput)
	if result.ChangesWritten {
		if err := allianceStore.WriteSnapshot(); err != nil {
			editorName := discordutil.GetInteractionAuthor(i).Username
			fmt.Printf("\nDEBUG | Changes written during alliances multi update. Editor: %s\n", editorName)

			return fmt.Errorf("error writing alliances after multi update. failed to write snapshot\n%v", err)
		}
	}

	//#region Build info output messages
	parts := sets.New[string]()

	removedNations := sets.New[string]()
	removedAlliances := sets.New[string]()
	for a, nations := range result.RemovedFrom {
		removedAlliances.Append(a)
		for _, n := range nations {
			removedNations.Append(n)
		}
	}
	if len(removedNations) > 0 {
		parts.Append(fmt.Sprintf(
			"✅ Removed nation(s): ```%s``` from alliance(s): ```%s```",
			strings.Join(removedNations.Keys(), ", "),
			strings.Join(removedAlliances.Keys(), ", "),
		))
	}

	addedNations := sets.New[string]()
	addedAlliances := sets.New[string]()
	for a, nations := range result.AddedTo {
		addedAlliances.Append(a)
		for _, n := range nations {
			addedNations.Append(n)
		}
	}
	if len(addedNations) > 0 {
		parts.Append(fmt.Sprintf(
			"✅ Added nation(s): ```%s``` to alliance(s): ```%s```",
			strings.Join(addedNations.Keys(), ", "),
			strings.Join(addedAlliances.Keys(), ", "),
		))
	}

	if len(result.InvalidAlliances) > 0 {
		parts.Append(fmt.Sprintf(
			"⚠️ Invalid/non-existent alliance(s):```%s```",
			strings.Join(result.InvalidAlliances.Keys(), ", "),
		))
	}
	if len(result.InvalidNations) > 0 {
		parts.Append(fmt.Sprintf(
			"⚠️ Invalid/non-existent nation(s):```%s```",
			strings.Join(result.InvalidNations.Keys(), ", "),
		))
	}
	// if len(result.Violations) > 0 {
	// 	parts = append(parts, fmt.Sprintf("Alliances with <2 nations:```%s```", strings.Join(result.Violations, ", ")))
	// }
	for allianceName, names := range result.AlreadyPuppets {
		parts.Append(fmt.Sprintf(
			"ℹ️ Cannot add nations to `%s` that are already puppets:```%s```",
			allianceName, strings.Join(names, ", "),
		))
	}

	if len(parts) > 0 {
		_, err := discordutil.FollowupContentEphemeral(s, i, strings.Join(parts.Keys(), "\n\n"))
		return err
	}
	//#endregion

	_, err = discordutil.FollowupContentEphemeral(s, i, "No changes made. Stop wasting my computing resources ya pleb.")
	return err
}

// TODO: Create a scheduled job that loops through alliances, removing nations that no longer exist.
// TODO: Make this func use MultiUpdateAlliances if possible.
// /alliance update nations
func handleAllianceEditorModalNationsUpdate(
	s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance,
	allianceStore *store.Store[database.Alliance],
) error {
	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	// build map of Name -> NationInfo for O(1) lookups
	nationByName := nationStore.EntriesFunc(func(n oapi.NationInfo) string {
		return strings.ToLower(n.Name)
	})

	// start with a set of existing UUIDs for easier add/remove
	nationUUIDs := utils.CopyMap(alliance.OwnNations)

	puppetAlliances := alliance.ChildAlliances(allianceStore.Values())
	puppetNationUUIDs := puppetAlliances.NationIds()

	inputs := discordutil.GetModalInputs(i)
	removeInput := strings.TrimSpace(inputs["remove"])
	addInput := strings.TrimSpace(inputs["add"])

	var notAdded, notRemoved, alreadyPuppets []string
	if removeInput != "" {
		removeNames, _ := utils.ParseFieldsStr(removeInput, ',')
		for _, name := range removeNames {
			n, ok := nationByName[strings.ToLower(name)]
			if !ok {
				// Can't remove dis input name bc it isn't even a nation.
				notRemoved = append(notRemoved, name)
				continue
			}

			// If it isn't already present, this is just a safe no-op.
			delete(nationUUIDs, n.UUID)
		}
	}
	if addInput != "" {
		addNames, _ := utils.ParseFieldsStr(addInput, ',')
		for _, name := range addNames {
			n, ok := nationByName[strings.ToLower(name)]
			if !ok {
				// Can't add dis input name bc it isn't even a nation.
				notAdded = append(notAdded, name)
				continue
			}

			if isPuppet := puppetNationUUIDs.Has(n.UUID); isPuppet {
				alreadyPuppets = append(alreadyPuppets, name)
				continue
			}

			nationUUIDs.Append(n.UUID)
		}
	}

	allNationsAmt := len(nationUUIDs) + len(puppetNationUUIDs)
	if allNationsAmt < 2 {
		return fmt.Errorf("An alliance cannot have a single nation or no nations!\n" +
			"There must be a total of two nations either directly, via puppet alliances, or both.",
		)
	}

	//#region Build info output messages
	var messages []string
	if len(notRemoved) > 0 {
		messages = append(messages, fmt.Sprintf(
			"The following nations were not removed as they do not exist:```%s```",
			strings.Join(notRemoved, ", "),
		))
	}
	if len(notAdded) > 0 {
		messages = append(messages, fmt.Sprintf(
			"The following nations were not added as they do not exist:```%s```",
			strings.Join(notAdded, ", "),
		))
	}
	if len(alreadyPuppets) > 0 {
		messages = append(messages, fmt.Sprintf(
			"The following nations were skipped as they are already puppets:```%s```",
			strings.Join(alreadyPuppets, ", "),
		))
	}
	//#endregion

	if len(messages) < 1 && utils.MapKeysEqual(alliance.OwnNations, nationUUIDs) {
		return fmt.Errorf("Alliance not edited. No changes were made to the nation list.")
	}

	// Update alliance with new nation list
	alliance.OwnNations = nationUUIDs
	alliance.SetUpdated()

	// Persist changes
	allianceStore.Set(strings.ToLower(alliance.Identifier), *alliance)
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	content := "Successfully edited alliance. Result:"
	embed := embeds.NewAllianceEmbed(s, allianceStore, *alliance, nil)
	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	//#region Send feedback messages if any
	if len(messages) > 0 {
		discordutil.FollowupContentEphemeral(s, i, strings.Join(messages, "\n"))
	}
	//#endregion

	return nil
}

// /alliance update leaders
func handleAllianceEditorModalLeadersUpdate(
	s *discordgo.Session, i *discordgo.Interaction,
	alliance *database.Alliance, allianceStore *store.Store[database.Alliance],
) error {
	playerStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.PLAYERS_STORE)
	if err != nil {
		return fmt.Errorf("error updating leaders for alliance: %s. failed to get player store from DB", alliance.Identifier)
	}

	// build map of Name -> BasicPlayer for O(1) lookups
	playerByName := playerStore.EntriesFunc(func(p database.BasicPlayer) string {
		return strings.ToLower(p.Name)
	})

	// start with a set of existing UUIDs for easier add/remove
	leaderUUIDs := utils.CopyMap(alliance.Optional.Leaders)

	var notAdded, notRemoved []string
	var inputs = discordutil.GetModalInputs(i)

	if strings.TrimSpace(inputs["remove"]) != "" {
		removeNames, _ := utils.ParseFieldsStr(inputs["remove"], ',')
		for _, name := range removeNames {
			p, ok := playerByName[strings.ToLower(name)]
			if !ok {
				notRemoved = append(notRemoved, name)
				continue // Can't remove dis player cuz they dont exist cuh
			}

			// remove leader if dey exist
			if _, exists := leaderUUIDs[p.UUID]; exists {
				delete(leaderUUIDs, p.UUID)
			} else {
				notRemoved = append(notRemoved, name)
			}
		}
	}
	if strings.TrimSpace(inputs["add"]) != "" {
		addNames, _ := utils.ParseFieldsStr(inputs["add"], ',')
		for _, name := range addNames {
			p, ok := playerByName[strings.ToLower(name)]
			if !ok {
				notAdded = append(notAdded, name)
				continue
			}

			leaderUUIDs[p.UUID] = struct{}{}
		}
	}

	// Update alliance with new nation list
	alliance.Optional.Leaders = leaderUUIDs // TODO: Keep as igns, then use alliance.SetLeaders
	alliance.SetUpdated()

	// Persist changes
	allianceStore.Set(strings.ToLower(alliance.Identifier), *alliance)
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	content := "Successfully edited alliance. Result:"
	embed := embeds.NewAllianceEmbed(s, allianceStore, *alliance, nil)
	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	//#region Build & send feedback message
	var messages []string
	if len(notRemoved) > 0 {
		messages = append(messages, fmt.Sprintf(
			"The following leaders were not removed as they are not present:```%s```",
			strings.Join(notRemoved, ", "),
		))
	}
	if len(notAdded) > 0 {
		messages = append(messages, fmt.Sprintf(
			"The following leaders were not added as they do not exist:```%s```",
			strings.Join(notAdded, ", "),
		))
	}

	if len(messages) > 0 {
		discordutil.FollowupContentEphemeral(s, i, strings.Join(messages, "\n"))
	}
	//#endregion

	return nil
}

func handleAllianceEditorModalFunctional(
	s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance,
	allianceStore *store.Store[database.Alliance],
) error {
	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	inputs := discordutil.GetModalInputs(i)

	oldIdent := alliance.Identifier
	ident := utils.DefaultIfEmpty(inputs["identifier"], oldIdent)
	label := utils.DefaultIfEmpty(inputs["label"], alliance.Label)

	parentIdent := alliance.Parent
	parentInput := strings.ReplaceAll(inputs["parent"], " ", "")
	parentInputLower := strings.ToLower(parentInput)

	if slices.Contains(REMOVE_KEYWORDS, parentInputLower) {
		parentIdent = nil
	} else if parentInput != "" {
		parent, err := allianceStore.Get(parentInputLower)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Parent alliance `%s` does not exist.", parentInput),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		parentIdent = &parent.Identifier
	}

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

	nationUUIDs := alliance.OwnNations
	missingNations := []string{}

	// We have some input, meaning nations are being edited, not staying as previous value.
	if inputs["nations"] != "" {
		inputNations := strings.Split(strings.ReplaceAll(inputs["nations"], " ", ""), ",")

		//#region Check nations name inputs are valid and grab their UUIDs.
		validNations, missing := validateNations(nationStore, inputNations)
		if len(validNations) < 1 {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Could not edit alliance `%s`.\nNone of the input nation names were valid nations.\n", oldIdent),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		missingNations = missing
		nationUUIDs = lo.Associate(validNations, func(n oapi.NationInfo) (string, struct{}) {
			return n.UUID, struct{}{}
		})
		//#endregion
	}

	// Update after all validation/transformations complete.
	alliance.Identifier = ident
	alliance.Label = label
	alliance.RepresentativeID = representativeID
	alliance.OwnNations = nationUUIDs
	alliance.Parent = parentIdent

	alliance.SetUpdated()

	// Update store
	allianceStore.Set(strings.ToLower(ident), *alliance)
	if !strings.EqualFold(oldIdent, ident) {
		allianceStore.Delete(strings.ToLower(oldIdent)) // remove old key if identifier changed
	}

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	embed := embeds.NewAllianceEmbed(s, allianceStore, *alliance, nil)
	content := "Successfully edited alliance. Result:"
	if len(missingNations) > 0 {
		embed.Color = discordutil.GOLD
		content = fmt.Sprintf(
			"Partially edited alliance, the following nations were invalid:```%s```",
			strings.Join(missingNations, ", "),
		)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	return nil
}

func handleAllianceEditorModalOptional(
	s *discordgo.Session, i *discordgo.Interaction,
	alliance *database.Alliance, allianceStore *store.Store[database.Alliance],
) error {
	inputs := discordutil.GetModalInputs(i)

	//#region Leaders validation
	playerStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.PLAYERS_STORE)
	if err != nil {
		return fmt.Errorf("error updating leaders for alliance: %s. failed to get player store from DB", alliance.Identifier)
	}

	invalidLeaders := []string{}
	if strings.TrimSpace(inputs["leaders"]) != "" {
		leaders, err := utils.ParseFieldsStr(inputs["leaders"], ',')
		invalidLeaders, err = alliance.SetLeaders(playerStore, leaders...)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("An error occurred while setting alliance leaders:```%s```", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}
	}
	//#endregion

	//#region Type validation
	inputType := strings.TrimSpace(inputs["type"])
	if inputType != "" {
		// do not allow empty string or bogus amogus to be matched.
		// keep using the old value of Type in that case.
		switch strings.ToLower(inputType) {
		case "mega", "meganation":
			alliance.Type = database.AllianceTypeMeganation
		case "org", "organisation", "organization":
			alliance.Type = database.AllianceTypeOrganisation
		case "null", "none":
			alliance.Type = database.AllianceTypePact
		}
	}
	//#endregion

	//#region Image validation
	image := strings.TrimSpace(inputs["image"])
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
	//#endregion

	//#region Discord invite validation
	var discordCode string
	if alliance.Optional.DiscordCode != nil {
		discordCode = *alliance.Optional.DiscordCode
	}

	discordInput := strings.TrimSpace(inputs["discord"])
	if discordInput != "" {
		code, err := discordutil.ExtractInviteCode(discordInput)
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
	//#endregion

	//#region Colours validation
	var fillColour, outlineColour string
	if alliance.Optional.Colours != nil {
		if alliance.Optional.Colours.Fill != nil {
			fillColour = *alliance.Optional.Colours.Fill
		}
		if alliance.Optional.Colours.Outline != nil {
			outlineColour = *alliance.Optional.Colours.Outline
		}
	}

	colours := strings.TrimSpace(strings.ReplaceAll(inputs["colours"], "#", ""))
	if colours != "" {
		var spaceSeperated bool

		fillColour, outlineColour, spaceSeperated = strings.Cut(colours, " ")
		fillColour = strings.TrimSpace(fillColour)
		if spaceSeperated {
			outlineColour = strings.TrimSpace(outlineColour)
		} else {
			outlineColour = fillColour // Use fill colour for outline if only one provided.
		}

		ok := utils.ValidateHexColour(fillColour)
		if !ok {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: "Input for field **Colours** contains an invalid HEX code as the fill colour.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		ok = utils.ValidateHexColour(outlineColour)
		if !ok {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: "Input for field **Colours** contains an invalid HEX code as the outline colour.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}
	}
	//#endregion

	// Update alliance fields after all validation/transformations complete.
	if discordCode != "" {
		alliance.Optional.DiscordCode = &discordCode
	}

	if image != "" {
		alliance.Optional.ImageURL = &image
	}

	// If both colours are invalid or unspecified, Optional.Colours will remain nil
	if fillColour != "" || outlineColour != "" {
		alliance.Optional.Colours = &database.AllianceColours{}

		if fillColour != "" {
			alliance.Optional.Colours.Fill = &fillColour
		}
		if outlineColour != "" {
			alliance.Optional.Colours.Outline = &outlineColour
		}
	}

	// Update alliance in store
	alliance.SetUpdated()
	allianceStore.Set(strings.ToLower(alliance.Identifier), *alliance)

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully edited alliance. Result:",
		Embeds: []*discordgo.MessageEmbed{
			embeds.NewAllianceEmbed(s, allianceStore, *alliance, nil),
		},
	})

	// After sending updated alliance embed, report missing leaders if any.
	if len(invalidLeaders) > 0 {
		discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf(
			"The following leaders do not exist and were not included:```%s```",
			strings.Join(invalidLeaders, ", "),
		))
	}

	return nil
}

func MultiUpdateAlliances(
	allianceStore *store.Store[database.Alliance],
	nationStore *store.Store[oapi.NationInfo],
	additions map[string][]string, // alliance identifier → nations to add
	removals map[string][]string, // alliance identifier → nations to remove
) MultiUpdateResult {
	result := MultiUpdateResult{
		AddedTo:          make(map[string][]string),
		RemovedFrom:      make(map[string][]string),
		AlreadyPuppets:   make(map[string][]string),
		InvalidAlliances: sets.New[string](),
		InvalidNations:   sets.New[string](),
	}

	// Build lookup maps
	nationByName := nationStore.EntriesFunc(func(n oapi.NationInfo) string { return strings.ToLower(n.Name) })

	alliances := allianceStore.Values()
	allianceByIdent := lo.Associate(alliances, func(a database.Alliance) (string, database.Alliance) {
		return strings.ToLower(a.Identifier), a
	})

	// ----- REMOVALS -----
	for allianceIdent, nationNames := range removals {
		a, ok := allianceByIdent[strings.ToLower(allianceIdent)]
		if !ok {
			result.InvalidAlliances.Append(allianceIdent)
			continue
		}

		removed := []string{}
		nationUUIDs := utils.CopyMap(a.OwnNations)
		for _, name := range nationNames {
			n, ok := nationByName[strings.ToLower(name)]
			if !ok {
				result.InvalidNations.Append(name)
				continue
			}
			if exists := nationUUIDs.Has(n.UUID); exists {
				delete(nationUUIDs, n.UUID)
				removed = append(removed, n.Name)
			}
		}
		if len(removed) == 0 {
			continue
		}

		a.OwnNations = nationUUIDs
		a.SetUpdated()
		allianceStore.Set(strings.ToLower(a.Identifier), a)

		result.RemovedFrom[a.Identifier] = removed
		result.ChangesWritten = true
	}

	// ----- ADDITIONS -----
	for allianceIdent, nationNames := range additions {
		a, ok := allianceByIdent[strings.ToLower(allianceIdent)]
		if !ok {
			result.InvalidAlliances.Append(allianceIdent)
			continue
		}

		nationUUIDs := utils.CopyMap(a.OwnNations)
		puppetUUIDs := a.ChildAlliances(alliances).NationIds()

		var addedNames, alreadyPuppetNames []string
		for _, name := range nationNames {
			n, ok := nationByName[strings.ToLower(name)]
			if !ok {
				result.InvalidNations.Append(name)
				continue
			}

			if puppetUUIDs.Has(n.UUID) {
				alreadyPuppetNames = append(alreadyPuppetNames, n.Name)
				continue
			}
			if nationUUIDs.Has(n.UUID) {
				continue
			}

			nationUUIDs.Append(n.UUID)
			addedNames = append(addedNames, n.Name)
		}
		if len(alreadyPuppetNames) > 0 {
			result.AlreadyPuppets[a.Identifier] = alreadyPuppetNames
		}
		if len(addedNames) == 0 {
			continue // nothing to add. no need to reflect any changes
		}

		a.OwnNations = nationUUIDs
		a.SetUpdated()
		allianceStore.Set(strings.ToLower(a.Identifier), a)

		result.AddedTo[a.Identifier] = addedNames
		result.ChangesWritten = true
	}

	return result
}

func generateAllianceID() (id uint64, createdTs uint64) {
	createdTs = uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))       // Safe to cast to uint since Intn returns 0-n anyway.
	return (createdTs << 16) | suffix, createdTs
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

	ext := strings.ToLower(path.Ext(strings.TrimRight(u.Path, "/")))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
	default:
		return "", errors.New("url does not point to an image type")
	}

	return u.String(), nil
}

func validateNations(nationStore *store.Store[oapi.NationInfo], input []string) (valid []oapi.NationInfo, missing []string) {
	if len(input) == 0 {
		return nil, nil // in case we were stupid and didn't provide an input
	}

	nameMap := nationStore.EntriesFunc(func(n oapi.NationInfo) string {
		return strings.ToLower(n.Name)
	})

	for _, name := range input {
		if n, ok := nameMap[strings.ToLower(name)]; ok {
			valid = append(valid, n)
		} else {
			missing = append(missing, name)
		}
	}

	return
}
