package slashcommands

import (
	"bytes"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

const EDITOR_ROLE = "966359842417705020"
const SR_EDITOR_ROLE = "1143253762039873646"
const ALLIANCE_BACKUP_CHANNEL = "1438592337335947314"

var REMOVE_KEYWORDS = []string{"null", "none", "remove", "delete"}

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
			Description: "Query information about an alliance (meganation, organisation or pact).",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "Sends a paginator enabling navigation through all registered alliances.",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance. This command is for Editors only.",
		},
		{
			// TODO: Maybe turn into modal with text inputs "Identifier" and "Disband Reason".
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "disband",
			Description: "Disband an alliance. This command is for Editors only.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:        "edit",
			Description: "Edit alliance info. This command is for Editors only.",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "functional",
					Description: "Edit alliance fields that are required for basic functionality.",
					Options: AppCommandOpts{
						discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "optional",
					Description: "Edit alliance fields that are not tied to its functionality.",
					Options: AppCommandOpts{
						discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
					},
				},
			},
		},
	}
}

func (cmd AllianceCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()

	opt := cdata.GetOption("query")
	if opt != nil {
		err := discordutil.DeferReply(s, i.Interaction)
		if err != nil {
			return err
		}

		return queryAlliance(s, i.Interaction, cdata)
	}

	if opt = cdata.GetOption("list"); opt != nil {
		return listAlliances(s, i.Interaction)
	}

	if opt = cdata.GetOption("create"); opt != nil {
		return createAlliance(s, i.Interaction)
	}
	if opt = cdata.GetOption("disband"); opt != nil {
		return disbandAlliance(s, i.Interaction, cdata)
	}
	if opt = cdata.GetOption("edit"); opt != nil {
		return editAlliance(s, i.Interaction, cdata)
	}

	return nil
}

func (cmd AllianceCommand) HandleModal(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	if customID == "alliance_creator" {
		err := handleAllianceCreatorModal(s, i)
		if err != nil {
			discordutil.ReplyWithError(s, i, err)
			return err
		}
	}

	if strings.HasPrefix(customID, "alliance_editor") {
		allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
		if err != nil {
			return err
		}

		ident := strings.Split(customID, "@")[1]
		alliance, err := allianceStore.GetKey(strings.ToLower(ident))
		if err != nil {
			//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

			discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
			return err
		}

		if strings.HasPrefix(customID, "alliance_editor_functional") {
			err := handleAllianceEditorModalFunctional(s, i, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i, err)
				return err
			}
		}

		if strings.Contains(customID, "alliance_editor_optional") {
			err := handleAllianceEditorModalOptional(s, i, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i, err)
				return err
			}
		}
	}

	return nil
}

func queryAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	ident := cdata.GetOption("query").GetOption("identifier").StringValue()
	alliance, err := allianceStore.GetKey(strings.ToLower(ident))
	if err != nil {
		//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

		_, err := discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	_, err = discordutil.FollowupEmbeds(s, i, shared.NewAllianceEmbed(s, alliance, allianceStore))
	return err
}

func listAlliances(s *discordgo.Session, i *discordgo.Interaction) error {
	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	alliances := allianceStore.Values()

	allianceCount := len(alliances)
	if allianceCount == 0 {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No alliances seem to exist? Something may have gone wrong with the database or alliance store.",
		})

		return err
	}

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i, allianceCount, perPage)
	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(allianceCount)

		allianceStrings := []string{}
		for idx, item := range alliances[start:end] {
			allianceName := item.Identifier
			if item.Optional.DiscordCode == nil {
				allianceName += fmt.Sprintf(" / %s", item.Label)
			} else {
				allianceName = fmt.Sprintf(
					"[%s / %s](https://discord.gg/%s)",
					item.Identifier,
					item.Label,
					*item.Optional.DiscordCode,
				)
			}

			leaderStr := "`Unknown/Error`"
			leaders, err := item.GetLeaderNames()
			if err != nil {
				fmt.Printf("%s an error occurred getting leaders for alliance %s:\n%v", time.Now().Format(time.Stamp), item.Identifier, err)
			} else {
				leaderStr = strings.Join(lo.Map(leaders, func(leader string, _ int) string {
					return fmt.Sprintf("`%s`", leader)
				}), ", ")
			}
			representativeName := "`Unknown/Error`"
			repUser, err := s.User(*item.RepresentativeID)
			if err == nil {
				representativeName = repUser.Username
			}

			nations, towns, residents, area, wealth := item.GetStats(nationStore, allianceStore)
			allianceStrings = append(allianceStrings, fmt.Sprintf(
				"%d. %s (%s)\nLeader(s): %s\nRepresentative: `%s`\nNations: %s\nTowns: %s\nResidents: %s\nSize: %s", start+idx+1,
				allianceName, item.Type.Colloquial(), leaderStr, representativeName,
				utils.HumanizedSprintf("`%d`", len(nations)),
				utils.HumanizedSprintf("`%d`", len(towns)),
				utils.HumanizedSprintf("`%d`", residents),
				utils.HumanizedSprintf("`%d` %s (Worth `%d` %s)", area, shared.EMOJIS.CHUNK, wealth, shared.EMOJIS.GOLD_INGOT),
			))
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("[%d] List of Alliances | %s", allianceCount, pageStr),
			Description: strings.Join(allianceStrings, "\n\n"),
			Color:       discordutil.DARK_AQUA,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
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
				Label:       "Alliance Name (4-36 chars)",
				Placeholder: "Enter this alliance's full name.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
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
				Style:       discordgo.TextInputParagraph,
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

	a, _ := allianceStore.FindFirst(func(a database.Alliance) bool {
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

	allianceStore.DeleteKey(strings.ToLower(a.Identifier))

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

func editAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	isEditor, _ := discordutil.HasRole(i.Member, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	opt := cdata.GetOption("edit")
	functional := opt.GetOption("functional")
	optional := opt.GetOption("optional")
	if functional != nil {
		opt = functional
	} else {
		opt = optional
	}

	ident := opt.GetOption("identifier").StringValue()
	alliance, err := allianceStore.GetKey(strings.ToLower(ident))
	if err != nil {
		//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	if functional != nil {
		return openEditorModalFunctional(s, i, alliance)
	}

	return openEditorModalOptional(s, i, alliance)
}

func openEditorModalFunctional(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	nationStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	nations := alliance.GetOwnNations(nationStore)
	nationNames := lo.Map(nations, func(n oapi.NationInfo, _ int) string {
		return n.Name
	})

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
				Label:       "Alliance Name (4-36 chars)",
				Placeholder: alliance.Label,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
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
				Style:       discordgo.TextInputParagraph,
			}),
		},
	})
}

func openEditorModalOptional(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	discordPlaceholder := "Enter an invite link or code to the alliance's Discord."
	if alliance.Optional.DiscordCode != nil {
		discordPlaceholder = fmt.Sprintf("https://discord.gg/%s", *alliance.Optional.DiscordCode)
	}

	imagePlaceholder := "Enter the URL of the alliance's image/flag from the flags channel."
	if alliance.Optional.ImageURL != nil {
		imagePlaceholder = *alliance.Optional.ImageURL
	}

	leaderPlaceholder := "Enter the Minecraft IGNs of the alliance leaders, comma-separated."
	if alliance.Optional.Leaders != nil {
		leaderNames, err := alliance.GetLeaderNames()
		if err == nil {
			leaderPlaceholder = strings.Join(leaderNames, ", ")
		}
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

func handleAllianceEditorModalFunctional(
	s *discordgo.Session, i *discordgo.Interaction,
	alliance *database.Alliance, allianceStore *store.Store[database.Alliance],
) error {
	nationStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	inputs := discordutil.GetModalInputs(i)

	oldIdent := alliance.Identifier

	ident := defaultIfEmpty(inputs["identifier"], oldIdent)
	label := defaultIfEmpty(inputs["label"], alliance.Label)

	parent := alliance.Parent
	parentInput := strings.ReplaceAll(inputs["parent"], " ", "")
	parentInputLower := strings.ToLower(parentInput)

	if slices.Contains(REMOVE_KEYWORDS, parentInputLower) {
		parent = nil
	} else if parentInput != "" {
		pa, err := allianceStore.GetKey(parentInputLower)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Parent alliance `%s` does not exist.", parentInput),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		parent = &pa.Identifier
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

		//#region Nation name inputs -> UUIDs
		validNations, missing := validateNations(nationStore, inputNations)
		if len(validNations) < 1 {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Could not edit alliance `%s`.\nNone of the input nation names were valid nations.\n", oldIdent),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}

		missingNations = missing
		nationUUIDs = lo.Map(validNations, func(n oapi.NationInfo, _ int) string {
			return n.UUID
		})
		//#endregion
	}

	// Update after all validation/transformations complete.
	alliance.Identifier = ident
	alliance.Label = label
	alliance.RepresentativeID = representativeID
	alliance.OwnNations = nationUUIDs
	alliance.Parent = parent

	// Update store
	allianceStore.SetKey(strings.ToLower(ident), *alliance)
	if !strings.EqualFold(oldIdent, ident) {
		allianceStore.DeleteKey(strings.ToLower(oldIdent)) // remove old key if identifier changed
	}

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	embed := shared.NewAllianceEmbed(s, alliance, allianceStore)
	content := "Successfully edited alliance:"
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

	//#region Discord invite validation
	var discordCode string
	if alliance.Optional.DiscordCode != nil {
		discordCode = *alliance.Optional.DiscordCode
	}

	if discordInput := inputs["discord"]; discordInput != "" {
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

	//#region Leaders validation
	leadersInput := strings.ReplaceAll(inputs["leaders"], " ", "")

	invalid := []string{}
	if leadersInput != "" {
		var err error
		leaders := strings.Split(leadersInput, ",")

		invalid, err = alliance.SetLeaders(leaders...)
		if err != nil {
			discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("An error occurred while setting alliance leaders:```%s```", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			})

			return nil
		}
	}
	//#endregion

	// Update alliance fields after all validation/transformations complete.
	alliance.Type = database.NewAllianceType(inputs["type"]) // invalid input will default to pact
	alliance.Optional.DiscordCode = &discordCode
	alliance.Optional.ImageURL = &image
	alliance.Optional.Colours = &database.AllianceColours{
		Fill:    &fillColour,
		Outline: nil,
	}
	if outlineColour != "" {
		alliance.Optional.Colours.Outline = &outlineColour
	}

	// Update alliance in store
	allianceStore.SetKey(strings.ToLower(alliance.Identifier), *alliance)

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err := allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully edited alliance:",
		Embeds: []*discordgo.MessageEmbed{
			shared.NewAllianceEmbed(s, alliance, allianceStore),
		},
	})

	if len(invalid) > 0 {
		discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf(
			"The following leaders do not exist and were not included:```%s```",
			strings.Join(invalid, ", "),
		))
	}

	return nil
}

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

	//#region Nation name inputs -> UUIDs
	validNations, missingNations := validateNations(nationStore, inputNations)
	if len(validNations) < 1 {
		discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not create alliance `%s`.\nNone of the input nation names were valid nations.\n", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return nil
	}

	nationUUIDs := lo.Map(validNations, func(n oapi.NationInfo, _ int) string {
		return n.UUID
	})
	//#endregion

	//#region Validate parent alliance
	var parent *string
	parentInput := strings.ReplaceAll(inputs["parent"], " ", "")
	if parentInput != "" {
		pa, err := allianceStore.GetKey(strings.ToLower(parentInput))
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

	alliance := database.Alliance{
		UUID:             generateAllianceID(),
		Identifier:       ident,
		Label:            strings.TrimSpace(inputs["label"]),
		RepresentativeID: &representativeUser.ID,
		OwnNations:       nationUUIDs,
		Parent:           parent,
		Type:             database.AllianceTypePact,
	}

	allianceStore.SetKey(strings.ToLower(ident), alliance)

	// We instantly write the data to the db to make sure the changes stick without waiting for graceful shutdown,
	// since the bot could panic and not recover at any moment and all changes would be lost.
	err = allianceStore.WriteSnapshot()
	if err != nil {
		return fmt.Errorf("error saving edited alliance '%s'. failed to write snapshot\n%v", alliance.Identifier, err)
	}

	embed := shared.NewAllianceEmbed(s, &alliance, allianceStore)
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

	//fmt.Print(utils.Prettify(alliance))
	return nil
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
	inputNations := make(map[string]struct{}, len(input))
	for _, name := range input {
		inputNations[strings.ToLower(name)] = struct{}{}
	}

	valid = nationStore.FindMany(func(n oapi.NationInfo) bool {
		_, ok := inputNations[strings.ToLower(n.Name)]
		return ok
	})

	// filter out all the valid and now unneeded ones.
	for _, n := range valid {
		delete(inputNations, strings.ToLower(n.Name))
	}

	return valid, slices.Collect(maps.Keys(inputNations))
}

func sendAllianceBackup(s *discordgo.Session, i *discordgo.Interaction, a *database.Alliance, reason string) {
	allianceJSON, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		fmt.Printf("could not send backup of alliance '%s'. failed to marshal\n%v", a.Identifier, err)
	}

	content := fmt.Sprintf("Alliance `%s` was %s by **%s**. A backup has been created:", a.Identifier, reason, i.Member.User)
	embedName := fmt.Sprintf("%s_%d.json", a.Identifier, a.CreatedTimestamp())

	_, err = s.ChannelFileSendWithMessage(ALLIANCE_BACKUP_CHANNEL, content, embedName, bytes.NewReader(allianceJSON))
	if err != nil {
		fmt.Printf("could not send backup of alliance '%s'. channel send message failed\n%v", a.Identifier, err)
	}
}
