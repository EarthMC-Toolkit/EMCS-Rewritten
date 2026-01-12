package slashcommands

import (
	"bytes"
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"encoding/json"
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

const EDITOR_ROLE = "966359842417705020"
const SR_EDITOR_ROLE = "1143253762039873646"
const ALLIANCE_BACKUP_CHANNEL = "1438592337335947314"

var REMOVE_KEYWORDS = []string{"null", "none", "remove", "delete"}

type AllianceCommand struct{}

func (cmd AllianceCommand) Name() string { return "alliance" }
func (cmd AllianceCommand) Description() string {
	return "Query a single alliance or navigate through existing alliances."
}

func (cmd AllianceCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about an alliance (meganation, organisation or pact).",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "Sends a paginator enabling navigation through all registered alliances.",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "score",
			Description: "Provides a breakdown of this alliance's score, which affects its rank.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance. EDITORS ONLY",
		},
		{
			// TODO: Maybe turn into modal with text inputs "Identifier" and "Disband Reason".
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "disband",
			Description: "Disband an alliance. EDITORS ONLY",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:        "update",
			Description: "Add/remove info to/from an alliance. EDITORS ONLY",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "nations",
					Description: "Add or remove nations from an alliance.",
					Options: AppCommandOpts{
						discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leaders",
					Description: "Add or remove leaders from an alliance.",
					Options: AppCommandOpts{
						discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
					},
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:        "edit",
			Description: "Bulk overwrite info of an alliance. EDITORS ONLY",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "functional",
					Description: "Edit alliance fields that are required for basic functionality.",
					Options: AppCommandOpts{
						discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "optional",
					Description: "Edit alliance fields that are not tied to its functionality.",
					Options: AppCommandOpts{
						discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
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

	if opt = cdata.GetOption("score"); opt != nil {
		err := discordutil.DeferReply(s, i.Interaction)
		if err != nil {
			return err
		}

		return queryAllianceScore(s, i.Interaction, cdata)
	}

	if opt = cdata.GetOption("list"); opt != nil {
		return listAlliances(s, i.Interaction)
	}

	if opt = cdata.GetOption("update"); opt != nil {
		return editAlliance(s, i.Interaction, opt)
	}
	if opt = cdata.GetOption("edit"); opt != nil {
		return editAlliance(s, i.Interaction, opt)
	}

	if opt = cdata.GetOption("create"); opt != nil {
		return createAlliance(s, i.Interaction)
	}
	if opt = cdata.GetOption("disband"); opt != nil {
		return disbandAlliance(s, i.Interaction, cdata)
	}

	return nil
}

// TODO: Getting store and looping through entries every filter/keypress could become costly?
func (cmd AllianceCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error {
	cdata := i.ApplicationCommandData()
	if len(cdata.Options) == 0 {
		return nil
	}

	// top-level sub cmd or group
	subCmd := cdata.Options[0]
	switch subCmd.Name {
	case "update":
		fallthrough
	case "edit":
		fallthrough
	case "disband":
		fallthrough
	case "score":
		fallthrough
	case "query":
		return allianceIdentifierAutocomplete(s, i, cdata)
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
		alliance, err := allianceStore.Get(strings.ToLower(ident))
		if err != nil {
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

		if strings.HasPrefix(customID, "alliance_editor_nations") {
			err := handleAllianceEditorModalNationsUpdate(s, i, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i, err)
				return err
			}
		}

		if strings.Contains(customID, "alliance_editor_leaders") {
			err := handleAllianceEditorModalLeadersUpdate(s, i, alliance, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i, err)
				return err
			}
		}
	}

	return nil
}

func allianceIdentifierAutocomplete(
	s *discordgo.Session, i *discordgo.Interaction,
	cdata discordgo.ApplicationCommandInteractionData,
) error {
	focused, ok := discordutil.GetFocusedValue[string](cdata.Options)
	if !ok {
		return fmt.Errorf("alliance autocomplete error: focused value could not be cast as string")
	}

	allianceStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	var matches []database.Alliance
	if strings.TrimSpace(focused) == "" {
		alliances := allianceStore.Values()

		// Sort alphabetically by Identifier.
		// TODO: Sort by alliance rank first instead. Need to cache ranks for that tho.
		slices.SortFunc(alliances, func(a, b database.Alliance) int {
			return cmp.Compare(strings.ToLower(a.Identifier), strings.ToLower(b.Identifier))
		})

		matches = alliances
	} else {
		keyLower := strings.ToLower(focused)
		matches = allianceStore.FindAll(func(a database.Alliance) bool {
			if a.Label != "" && strings.Contains(strings.ToLower(a.Label), keyLower) {
				return true
			}
			if a.Identifier != "" && strings.Contains(strings.ToLower(a.Identifier), keyLower) {
				return true
			}

			return false
		})
	}

	// truncate to Discord limit
	if len(matches) > discordutil.AUTOCOMPLETE_CHOICE_LIMIT {
		limit := min(len(matches), discordutil.AUTOCOMPLETE_CHOICE_LIMIT)
		matches = matches[:limit]
	}

	choices := discordutil.CreateAutocompleteChoices(matches, func(a database.Alliance, _ int) (string, string) {
		return fmt.Sprintf("%s | %s (%s)", a.Identifier, a.Label, a.Type.Colloquial()), a.Identifier
	})

	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

func queryAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	ident := cdata.GetOption("query").GetOption("identifier").StringValue()
	alliance, err := allianceStore.Get(strings.ToLower(ident))
	if err != nil {
		_, err := discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		fmt.Print(err)

		// Nations store failed, but we should still be able to send without rank info.
		_, err = discordutil.FollowupEmbeds(s, i, embeds.NewAllianceEmbed(s, allianceStore, *alliance, nil))
		return err
	}

	rankedAlliances := database.GetRankedAlliances(allianceStore, nationStore, database.DEFAULT_ALLIANCE_WEIGHTS)
	allianceRankInfo := rankedAlliances[alliance.UUID]

	_, err = discordutil.FollowupEmbeds(s, i, embeds.NewAllianceEmbed(s, allianceStore, *alliance, &allianceRankInfo))
	return err
}

func queryAllianceScore(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	ident := cdata.GetOption("score").GetOption("identifier").StringValue()
	alliance, err := allianceStore.Get(strings.ToLower(ident))
	if err != nil {
		_, err := discordutil.FollowupContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	WEIGHTS := database.DEFAULT_ALLIANCE_WEIGHTS

	rankedAlliances := database.GetRankedAlliances(allianceStore, nationStore, WEIGHTS)
	allianceRankInfo := rankedAlliances[alliance.UUID]

	residentsCalc := allianceRankInfo.Stats.Residents * WEIGHTS.Residents
	residentsStr := utils.HumanizedSprintf("Residents: `%.0f` * `%.1f` = **%.0f**",
		allianceRankInfo.Stats.Residents, WEIGHTS.Residents, residentsCalc,
	)

	nationsCalc := allianceRankInfo.Stats.Nations * WEIGHTS.Nations
	nationsStr := utils.HumanizedSprintf("Nations: `%.0f` * `%.1f` = **%.0f**",
		allianceRankInfo.Stats.Nations, WEIGHTS.Nations, nationsCalc,
	)

	townsCalc := allianceRankInfo.Stats.Towns * WEIGHTS.Towns
	townsStr := utils.HumanizedSprintf("Towns: `%.0f` * `%.1f` = **%.0f**",
		allianceRankInfo.Stats.Towns, WEIGHTS.Towns, townsCalc,
	)

	worthCalc := allianceRankInfo.Stats.Worth * WEIGHTS.Worth
	worthStr := utils.HumanizedSprintf("Worth: `%.0f` * `%.2f` = **%.0f**",
		allianceRankInfo.Stats.Worth, WEIGHTS.Worth, worthCalc,
	)

	scoreStr := utils.HumanizedSprintf("Total: `%.0f` + `%.0f` + `%.0f` + `%.0f` = **%.0f**",
		residentsCalc, nationsCalc, townsCalc, worthCalc, allianceRankInfo.Score,
	)

	// normalizedStr := utils.HumanizedSprintf("Normalized: `%.0f` / 2 = **%.0f**",
	// 	allianceRankInfo.Score*4, allianceRankInfo.Score,
	// )

	// TODO: Maybe include closest rival alliance and required score to surpass it.
	standingStr := utils.HumanizedSprintf("This alliance has a score of **%.0f** which places it at rank **%d** out of **%d**.",
		allianceRankInfo.Score, allianceRankInfo.Rank, allianceStore.Count(),
	)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Alliance Score Breakdown | `%s` | #%d", alliance.Identifier, allianceRankInfo.Rank),
		Description: fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n\n%s", standingStr,
			residentsStr, nationsStr, townsStr, worthStr,
			scoreStr, //normalizedStr,
		),
		Color:  discordutil.DARK_AQUA,
		Footer: embeds.DEFAULT_FOOTER,
	}

	_, err = discordutil.FollowupEmbeds(s, i, embed)
	return err
}

func listAlliances(s *discordgo.Session, i *discordgo.Interaction) error {
	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	allianceCount := allianceStore.Count()
	if allianceCount == 0 {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "No alliances seem to exist? Something may have gone wrong with the database or alliance store.",
		})

		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	entitiesStore, err := database.GetStore(mdb, database.ENTITIES_STORE)
	if err != nil {
		return err
	}

	reslist, _ := entitiesStore.Get("residentlist")
	townlesslist, _ := entitiesStore.Get("townlesslist")

	alliances := allianceStore.Values()

	rankedAlliances := database.GetRankedAlliances(allianceStore, nationStore, database.DEFAULT_ALLIANCE_WEIGHTS)
	slices.SortFunc(alliances, func(a, b database.Alliance) int {
		// sort alliances via rankedAlliances map. lowest (best) rank first
		return cmp.Compare(rankedAlliances[b.UUID].Rank, rankedAlliances[a.UUID].Rank)
	})

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i, allianceCount, perPage).
		WithTimeout(6 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(allianceCount)

		allianceStrings := []string{}
		for idx, a := range alliances[start:end] {
			allianceName := a.Identifier
			if a.Optional.DiscordCode == nil {
				allianceName += fmt.Sprintf(" / %s", a.Label)
			} else {
				allianceName = fmt.Sprintf(
					"[%s / %s](https://discord.gg/%s)",
					a.Identifier,
					a.Label,
					*a.Optional.DiscordCode,
				)
			}

			leaderStr := "`None`"
			leaders := a.GetLeaderNames(reslist, townlesslist)
			if err != nil {
				fmt.Printf("%s an error occurred getting leaders for alliance %s:\n%v", time.Now().Format(time.Stamp), a.Identifier, err)
				leaderStr = "`Unknown/Error`"
			} else {
				if len(leaders) > 0 {
					leaderStr = strings.Join(lo.Map(leaders, func(leader string, _ int) string {
						return fmt.Sprintf("`%s`", leader)
					}), ", ")
				}
			}

			representativeName := "`Unknown/Error`"
			repUser, err := s.User(*a.RepresentativeID)
			if err == nil {
				representativeName = repUser.Username
			}

			ownNations := nationStore.GetFromSet(a.OwnNations)

			childNationIds := a.ChildAlliances(alliances).NationIds()
			childNations := nationStore.GetFromSet(childNationIds)

			towns, residents, area, worth := a.GetStats(ownNations, childNations)
			allianceStrings = append(allianceStrings, fmt.Sprintf(
				"%d. %s (%s)\nLeader(s): %s\nRepresentative: `%s`\nNations: %s\nTowns: %s\nResidents: %s\nSize: %s", start+idx+1,
				allianceName, a.Type.Colloquial(), leaderStr, representativeName,
				utils.HumanizedSprintf("`%d`", len(childNations)+len(ownNations)),
				utils.HumanizedSprintf("`%d`", len(towns)),
				utils.HumanizedSprintf("`%d`", residents),
				utils.HumanizedSprintf("`%d` %s (Worth `%d` %s)", area, shared.EMOJIS.CHUNK, worth, shared.EMOJIS.GOLD_INGOT),
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

func editAlliance(s *discordgo.Session, i *discordgo.Interaction, opt *discordgo.ApplicationCommandInteractionDataOption) error {
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

	// get the sub cmd currently being used
	// TODO: Replace this bollocks with a func that searches for active subcmd.
	subCmd := opt
	if opt.Name == "edit" {
		functional := opt.GetOption("functional")
		if functional != nil {
			subCmd = functional
		} else {
			subCmd = opt.GetOption("optional")
		}
	}
	if opt.Name == "update" {
		nations := opt.GetOption("nations")
		if nations != nil {
			subCmd = nations
		} else {
			subCmd = opt.GetOption("leaders")
		}
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
	case "functional":
		return openEditorModalFunctional(s, i, alliance)
	case "optional":
		return openEditorModalOptional(s, i, alliance)
	case "nations":
		return openEditorModalNationsUpdate(s, i, alliance)
	case "leaders":
		return openEditorModalLeadersUpdate(s, i, alliance)
	}

	return fmt.Errorf("no valid option found for editAlliance")
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

// TODO: Create a scheduled job that loops through alliances, removing nations that no longer exist.
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

			if _, isPuppet := puppetNationUUIDs[n.UUID]; isPuppet {
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

	//#region Build feedback messages
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

func openEditorModalFunctional(s *discordgo.Session, i *discordgo.Interaction, alliance *database.Alliance) error {
	nationStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)
	nations := nationStore.GetFromSet(alliance.OwnNations)
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

	ident := defaultIfEmpty(inputs["identifier"], oldIdent)
	label := defaultIfEmpty(inputs["label"], alliance.Label)

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

	// Update alliance fields after all validation/transformations complete.
	alliance.Type = database.NewAllianceType(inputs["type"]) // invalid input will default to pact

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

	alliance.SetUpdated()

	// Update alliance in store
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

	//fmt.Print(utils.Prettify(alliance))
	return nil
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
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
