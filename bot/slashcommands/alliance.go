package slashcommands

import (
	"bytes"
	"cmp"
	"emcsrw/database"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/sets"
	"encoding/json"
	"fmt"
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
					Name:        "multi",
					Description: "Add/remove >=1 nation(s) from >=1 alliances at once.",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "nations",
					Description: "Add/remove >=1 nation(s) from a single alliance.",
					Options: AppCommandOpts{
						discordutil.AutocompleteStringOption("identifier", "The alliance's identifier/short name.", 3, 16, true),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leaders",
					Description: "Add/remove leaders from an alliance.",
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
		if err := discordutil.DeferReply(s, i.Interaction); err != nil {
			return err
		}

		return queryAlliance(s, i.Interaction, cdata)
	}

	if opt = cdata.GetOption("score"); opt != nil {
		if err := discordutil.DeferReply(s, i.Interaction); err != nil {
			return err
		}

		return queryAllianceScore(s, i.Interaction, cdata)
	}

	if opt = cdata.GetOption("list"); opt != nil {
		return listAlliances(s, i.Interaction)
	}

	if opt = cdata.GetOption("update"); opt != nil {
		return editAlliance(s, i.Interaction)
	}
	if opt = cdata.GetOption("edit"); opt != nil {
		return editAlliance(s, i.Interaction)
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

		if strings.EqualFold(customID, "alliance_editor_multi") {
			if err := discordutil.DeferReply(s, i); err != nil {
				return err
			}

			err := handleAllianceEditorModalMultiUpdate(s, i, allianceStore)
			if err != nil {
				discordutil.ReplyWithError(s, i, err)
			}

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
		// Sort alphabetically by Identifier.
		// TODO: Sort by alliance rank first instead. Need to cache ranks for that tho.
		alliances := allianceStore.ValuesSorted(func(a, b database.Alliance) int {
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

	alliancesRankInfo, _ := database.GetRankedAlliances(allianceStore, nationStore, database.DEFAULT_ALLIANCE_WEIGHTS)
	rankInfo := alliancesRankInfo[alliance.UUID]

	_, err = discordutil.FollowupEmbeds(s, i, embeds.NewAllianceEmbed(s, allianceStore, *alliance, &rankInfo))
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

	alliancesRankInfo, _ := database.GetRankedAlliances(allianceStore, nationStore, WEIGHTS)
	rankInfo := alliancesRankInfo[alliance.UUID]

	residentsCalc := rankInfo.Stats.Residents * WEIGHTS.Residents
	residentsStr := utils.HumanizedSprintf("Residents: `%.0f` * `%.1f` = **%.0f**",
		rankInfo.Stats.Residents, WEIGHTS.Residents, residentsCalc,
	)

	nationsCalc := rankInfo.Stats.Nations * WEIGHTS.Nations
	nationsStr := utils.HumanizedSprintf("Nations: `%.0f` * `%.1f` = **%.0f**",
		rankInfo.Stats.Nations, WEIGHTS.Nations, nationsCalc,
	)

	townsCalc := rankInfo.Stats.Towns * WEIGHTS.Towns
	townsStr := utils.HumanizedSprintf("Towns: `%.0f` * `%.1f` = **%.0f**",
		rankInfo.Stats.Towns, WEIGHTS.Towns, townsCalc,
	)

	worthCalc := rankInfo.Stats.Worth * WEIGHTS.Worth
	worthStr := utils.HumanizedSprintf("Worth: `%.0f` * `%.2f` = **%.0f**",
		rankInfo.Stats.Worth, WEIGHTS.Worth, worthCalc,
	)

	scoreStr := utils.HumanizedSprintf("Total: `%.0f` + `%.0f` + `%.0f` + `%.0f` = **%.0f**",
		residentsCalc, nationsCalc, townsCalc, worthCalc, rankInfo.Score,
	)

	// normalizedStr := utils.HumanizedSprintf("Normalized: `%.0f` / 2 = **%.0f**",
	// 	allianceRankInfo.Score*4, allianceRankInfo.Score,
	// )

	// TODO: Maybe include closest rival alliance and required score to surpass it.
	standingStr := utils.HumanizedSprintf("This alliance has a score of **%.0f** which places it at rank **%d** out of **%d**.",
		rankInfo.Score, rankInfo.Rank, allianceStore.Count(),
	)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Alliance Score Breakdown | `%s` | #%d", alliance.Identifier, rankInfo.Rank),
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

	alliancesRankInfo, alliances := database.GetRankedAlliances(allianceStore, nationStore, database.DEFAULT_ALLIANCE_WEIGHTS)
	slices.SortFunc(alliances, func(a, b database.Alliance) int {
		// sort alliances via rankedAlliances map. lowest (best) rank first
		return cmp.Compare(alliancesRankInfo[b.UUID].Rank, alliancesRankInfo[a.UUID].Rank)
	})

	// Init paginator with X items per page. Pressing a btn will change the current page and call PageFunc again.
	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i, allianceCount, perPage).
		WithTimeout(6 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(allianceCount)

		pageAlliances := alliances[start:end]
		allianceStrings := sets.Make[string](len(pageAlliances))
		for idx, a := range pageAlliances {
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
			allianceStrings.Append(fmt.Sprintf(
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
			Description: strings.Join(allianceStrings.Keys(), "\n\n"),
			Color:       discordutil.DARK_AQUA,
		}

		data.Embeds = []*discordgo.MessageEmbed{embed}
	}

	return paginator.Start()
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
