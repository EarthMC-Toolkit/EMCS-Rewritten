package embeds

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/sets"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/samber/lo/parallel"

	"github.com/bwmarrin/discordgo"
)

// NOTE: Potential import cycle. Consider just duplicating necessary funcs rather than importing discordutil.
var NewEmbedField = discordutil.NewEmbedField
var PrependField = discordutil.PrependField
var AddField = discordutil.AddField

var DEFAULT_FOOTER = &discordgo.MessageEmbedFooter{
	IconURL: "https://cdn.discordapp.com/avatars/263377802647175170/a_0cd469f208f88cf98941123eb1b52259.webp?size=512&animated=true",
	Text:    "Maintained by Owen3H • Open Source on GitHub 💛", // unless you maintain your own fork, pls keep this as is :)
}

// Returns a string listing player names with their affiliations.
// For example:
//
//	`Player1` of Town1 (**Nation1**)
//	`Player2` of Town2
//	`Player3` (Townless)
func GetAffiliationLines(players []database.BasicPlayer) string {
	if players == nil {
		return ""
	}

	lines := []string{}
	for _, p := range players {
		line := fmt.Sprintf("`%s`", p.Name)
		if p.Town != nil {
			line += fmt.Sprintf(" of %s", p.Town.Name)
			if p.Nation != nil {
				line += fmt.Sprintf(" (**%s**)", p.Nation.Name)
			}
		} else {
			line += " (Townless)"
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Creates a single embed showing info from the given Alliance.
func NewAllianceEmbed(
	s *discordgo.Session, allianceStore *store.Store[database.Alliance],
	a database.Alliance, rankInfo *database.AllianceRankInfo,
) *discordgo.MessageEmbed {
	playerStore, err := database.GetStoreForMap(shared.ACTIVE_MAP, database.PLAYERS_STORE)
	if err != nil {
		fmt.Printf("ERROR | Could not get player store for map %s:\n%v", shared.ACTIVE_MAP, err)
		return nil
	}

	// Leader field logic
	leadersValue := "`None`"
	leaders, err := a.GetLeaders(playerStore)
	if err != nil {
		fmt.Printf("ERROR | Could not get leaders for alliance %s:\n\t%v", a.Identifier, err)
	} else {
		leadersValue = GetAffiliationLines(leaders)
	}

	// Representative field logic
	representativeValue := "`None`"
	if a.RepresentativeID != nil {
		u, err := s.User(*a.RepresentativeID)
		if err == nil {
			representativeValue = fmt.Sprintf("`%s`", u.String())
		}
	}

	// Nation field logic
	nationStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.NATIONS_STORE)

	ownNations := nationStore.GetFromSet(a.OwnNations)
	ownNationNames := lo.Map(ownNations, func(n oapi.NationInfo, _ int) string {
		return n.Name
	})
	if len(ownNationNames) > 0 {
		slices.Sort(ownNationNames) // Alphabet sort
	}

	alliances := allianceStore.Values()
	childAlliances := a.ChildAlliances(alliances)
	childNations := nationStore.GetFromSet(childAlliances.NationIds())

	nationsAmt := len(ownNations) + len(childNations)
	towns, residentsAmt, area, wealth := a.GetStats(ownNations, childNations)
	stats := fmt.Sprintf("Towns: %s\nNations: %s\nResidents: %s\nSize: %s",
		utils.HumanizedSprintf("`%d`", len(towns)),
		utils.HumanizedSprintf("`%d`", nationsAmt),
		utils.HumanizedSprintf("`%d`", residentsAmt),
		utils.HumanizedSprintf("`%d` %s (Worth `%d` %s)", area, shared.EMOJIS.CHUNK, wealth, shared.EMOJIS.GOLD_INGOT),
	)

	registered := a.CreatedTimestamp() / 1000

	// Resort to default colour unless the alliance has a fill colour specified.
	embedColour := discordutil.DARK_AQUA
	colours := a.Optional.Colours
	if colours != nil && colours.Fill != nil {
		embedColour = utils.HexToInt(*colours.Fill)
	}

	title := fmt.Sprintf("Alliance Info | `%s`", a.Label)
	if rankInfo != nil {
		title += fmt.Sprintf(" | #%d", rankInfo.Rank)
	}

	embed := &discordgo.MessageEmbed{
		Color:  embedColour,
		Footer: DEFAULT_FOOTER,
		Title:  title,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Leader(s)", leadersValue, false),
			NewEmbedField("Stats", stats, true),
		},
	}

	var coloursStr = "No colours set."
	if a.Optional.Colours != nil && a.Optional.Colours.Fill != nil {
		fill := *a.Optional.Colours.Fill

		outline := fill
		if a.Optional.Colours.Outline != nil {
			outline = *a.Optional.Colours.Outline
		}

		coloursStr = fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", fill, outline)
	}

	AddField(embed, "Colours", coloursStr, true)
	AddField(embed, "Type", fmt.Sprintf("`%s`", a.Type.Colloquial()), true)
	AddField(embed, "Registered", fmt.Sprintf("<t:%d:f>\n<t:%d:R>", registered, registered), true)

	if a.UpdatedTimestamp != nil {
		updatedSec := *a.UpdatedTimestamp / 1000
		AddField(embed, "Last Updated", fmt.Sprintf("<t:%d:f>\n<t:%d:R>", updatedSec, updatedSec), true)
	}

	ownNationsValue := fmt.Sprintf("```%s```", strings.Join(ownNationNames, ", "))

	if len(childNations) < 1 {
		AddField(embed, fmt.Sprintf("Nations [%d]", len(ownNations)), ownNationsValue, false)
	} else {
		AddField(embed, fmt.Sprintf("Self Nations [%d]", len(ownNations)), ownNationsValue, false)

		childAllianceNames := lo.Map(childAlliances, func(a database.Alliance, _ int) string {
			return fmt.Sprintf("`%s`", a.Identifier)
		})

		childNationNames := lo.Map(childNations, func(n oapi.NationInfo, _ int) string {
			return n.Name
		})
		if len(childNationNames) > 0 {
			slices.Sort(childNationNames) // Alphabet sort
		}

		childNationsKey := fmt.Sprintf("Puppet Nations [%d]", len(childNations))
		childNationsValue := fmt.Sprintf(
			"Condensed list from `%d` puppet alliance(s): %s.```%s```",
			len(childAlliances),
			strings.Join(childAllianceNames, ", "),
			strings.Join(childNationNames, ", "),
		)

		AddField(embed, childNationsKey, childNationsValue, false)
	}

	AddField(embed, "Discord Representative", representativeValue, false)

	flag := a.Optional.ImageURL
	if flag != nil {
		if *flag != "" {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: *flag,
			}
		}
	}

	if a.Parent != nil {
		parentAlliance, err := allianceStore.Get(strings.ToLower(*a.Parent))
		if err == nil {
			embed.Description = fmt.Sprintf("*This alliance is a puppet of `%s` / `%s`*.", parentAlliance.Identifier, parentAlliance.Label)
		}
	}

	if a.Optional.DiscordCode != nil {
		embed.URL = fmt.Sprintf("https://discord.gg/%s", *a.Optional.DiscordCode)
		embed.Title = fmt.Sprintf("Alliance Info | %s", a.Label)
		if rankInfo != nil {
			embed.Title += fmt.Sprintf(" | #%d", rankInfo.Rank)
		}
	}

	return embed
}

// Builds an embed that describes a user with only minimal info.
//
// Should be preferred when the user is annoying and has opted-out of the Official API.
func NewBasicPlayerEmbed(player database.BasicPlayer, description string) *discordgo.MessageEmbed {
	townsStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
	town, _ := townsStore.Get(player.Town.UUID)

	townName := lo.Ternary(town == nil, "No Town", town.Name)
	nationName := lo.TernaryF(town.Nation.Name == nil,
		func() string { return "No Nation" },
		func() string { return *town.Nation.Name },
	)

	affiliation := "None (Townless)"
	if town != nil {
		spawn := town.Coordinates.Spawn
		affiliation = fmt.Sprintf("[%s](https://map.earthmc.net?x=%f&z=%f&zoom=3) (%s)", townName, spawn.X, spawn.Z, nationName)
	}

	title := fmt.Sprintf("Player Information | `%s`", player.Name)
	embed := &discordgo.MessageEmbed{
		Type:   discordgo.EmbedTypeRich,
		Color:  discordutil.DARK_PURPLE,
		Footer: DEFAULT_FOOTER,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", player.UUID),
		},
		Title:       title,
		Description: description,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Affiliation", affiliation, true),
		},
	}

	if town != nil {
		rank := "Resident"
		if town.Mayor.Name == player.Name {
			rank = "Mayor"
		}

		AddField(embed, "Rank", rank, true)
	}

	// TODO: Add town ranks using player.Town.Ranks

	AddField(embed, "Minecraft UUID", fmt.Sprintf("`%s`", player.UUID), false)

	return embed
}

// Builds an embed that describes a user using info from the Official API.
//
// Should be preferred when the user exists on said API and has not opted-out.
func NewPlayerEmbed(player oapi.PlayerInfo) *discordgo.MessageEmbed {
	registeredTs := player.Timestamps.Registered   // ms
	lastOnlineTs := player.Timestamps.LastOnline   // ms
	joinedTownTs := player.Timestamps.JoinedTownAt // ms

	status := "Offline" // Assume they are offline
	if player.Status.IsOnline {
		status = "Online"
	} else if lastOnlineTs != nil {
		status = fmt.Sprintf("Offline (Last Online: <t:%d:R>)", *lastOnlineTs/1000)
	}

	playerName := player.Name
	alias := playerName

	// Add Prefix if available
	if title := utils.CheckAlphanumeric(player.Title); title != "" {
		alias = fmt.Sprintf("%s %s", title, playerName)
	}

	// Add Postfix if available
	if surname := utils.CheckAlphanumeric(player.Surname); surname != "" {
		alias = fmt.Sprintf("%s %s", playerName, surname)
	}

	townName := lo.TernaryF(player.Town.Name == nil, func() string { return "No Town" }, func() string { return *player.Town.Name })
	nationName := lo.TernaryF(player.Nation.Name == nil, func() string { return "No Nation" }, func() string { return *player.Nation.Name })

	affiliation := "None (Townless)"
	if townName != "No Town" {
		townsStore, _ := database.GetStoreForMap(shared.ACTIVE_MAP, database.TOWNS_STORE)
		town, err := townsStore.Get(*player.Town.UUID)

		// Should never rly be false bc we established they aren't townless.
		if err == nil {
			spawn := town.Coordinates.Spawn
			affiliation = fmt.Sprintf("[%s](https://map.earthmc.net?x=%f&z=%f&zoom=3) (%s)", townName, spawn.X, spawn.Z, nationName)
		}
	}

	rank := "Resident"
	if player.Status.IsMayor {
		rank = "Mayor"
	}
	if player.Status.IsKing {
		rank = "Nation Leader"
	}

	friendsStr := "No Friends :("
	if player.Stats.NumFriends > 0 {
		friends := parallel.Map(player.Friends, func(e oapi.Entity, _ int) string { return e.Name })
		slices.Sort(friends)

		friendsStr = fmt.Sprintf("```%s```", strings.Join(friends, ", "))
	}

	title := fmt.Sprintf("Player Information | `%s`", playerName)
	if alias != playerName {
		// Alias differs from name (has surname and/or title)
		title += fmt.Sprintf(" aka \"%s\"", alias)
	}

	townRanks := player.Ranks.Town
	townRanksStr := "No ranks"
	if len(townRanks) > 0 {
		townRanksStr = strings.Join(lo.Map(townRanks, func(r string, _ int) string {
			return fmt.Sprintf("`%s`", r)
		}), ", ")
	}

	nationRanks, nationRanksStr := player.Ranks.Nation, "No ranks"
	if len(nationRanks) > 0 {
		nationRanksStr = strings.Join(lo.Map(nationRanks, func(r string, _ int) string {
			return fmt.Sprintf("`%s`", r)
		}), ", ")
	}

	embed := &discordgo.MessageEmbed{
		Type:   discordgo.EmbedTypeRich,
		Color:  discordutil.DARK_PURPLE,
		Footer: DEFAULT_FOOTER,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", player.UUID),
		},
		Title: title,
		Fields: []*discordgo.MessageEmbedField{
			// Affiliation (prepended)
			// Rank (prepended)
			NewEmbedField("Balance", utils.HumanizedSprintf("`%.0f`G %s", player.Stats.Balance, shared.EMOJIS.GOLD_INGOT), true),
			NewEmbedField("Status", status, true),
		},
	}

	if player.About != nil {
		about := *player.About
		if about != "" && about != shared.DEFAULT_ABOUT {
			embed.Description = fmt.Sprintf("*%s*", about)
		}
	}

	if joinedTownTs != nil {
		AddField(embed, "Joined Town", fmt.Sprintf("<t:%d:R>", *joinedTownTs/1000), true)
	}

	AddField(embed, "Registered", fmt.Sprintf("<t:%d:R>", registeredTs/1000), true)
	AddField(embed, "Appointed Ranks", fmt.Sprintf("Town: %s\nNation: %s", townRanksStr, nationRanksStr), false)
	AddField(embed, "Friends", friendsStr, false)
	AddField(embed, "Minecraft UUID", fmt.Sprintf("`%s`", player.UUID), false)

	// Second field
	if townName != "No Town" {
		PrependField(embed, "Rank", rank, true)
	}

	// First field
	PrependField(embed, "Affiliation", affiliation, true)

	return embed
}

func NewTownEmbed(town oapi.TownInfo) *discordgo.MessageEmbed {
	foundedTs := town.Timestamps.Registered / 1000 // Seconds

	townTitle := fmt.Sprintf("Town Information | `%s`", town.Name)
	// if town.Nation.Name != nil {
	// 	townTitle += fmt.Sprintf(" (%s)", *town.Nation.Name)
	// }

	colour := discordutil.GREEN
	if town.Status.Ruined {
		townTitle += " (Ruined)"
		colour = discordutil.DARK_GOLD
	}

	desc := ""
	if town.Board != shared.DEFAULT_TOWN_BOARD {
		desc = fmt.Sprintf("*%s*", town.Board)
	}

	overclaimShield := "`Inactive` " + shared.EMOJIS.SHIELD_RED
	if town.Status.HasOverclaimShield {
		overclaimShield = "`Active` " + shared.EMOJIS.SHIELD_GREEN
	}

	nationName := "No Nation"
	nationJoin := ""
	if town.Nation.Name != nil {
		nationName = *town.Nation.Name
		nationJoin = fmt.Sprintf(" (Joined <t:%d:R>)", *town.Timestamps.JoinedNationAt/1000)
	}

	spawn := town.Coordinates.Spawn

	balanceStr := utils.HumanizedSprintf("`%.0f`G %s", town.Bal(), shared.EMOJIS.GOLD_INGOT)
	residentsStr := utils.HumanizedSprintf("`%d`", town.Stats.NumResidents)
	trustedOutlawsStr := utils.HumanizedSprintf("`%d`/`%d`", town.Stats.NumTrusted, town.Stats.NumOutlaws)

	sizeStr := utils.HumanizedSprintf("`%d`/`%d` %s (Worth: `%d` %s)",
		town.Size(), town.MaxSize(),
		shared.EMOJIS.CHUNK, town.Worth(), shared.EMOJIS.GOLD_INGOT,
	)

	locationLink := fmt.Sprintf(
		"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)",
		spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
	)

	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       townTitle,
		Description: desc,
		Color:       colour,
		Footer:      DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Origin", fmt.Sprintf("Founded <t:%d:R> by `%s`", foundedTs, town.Founder), false),
			NewEmbedField("Mayor", fmt.Sprintf("`%s`", town.Mayor.Name), true),
			NewEmbedField("Nation", fmt.Sprintf("`%s`%s", nationName, nationJoin), true),
			NewEmbedField("Location", locationLink, true),
			NewEmbedField("Stats", fmt.Sprintf(
				"Size: %s\nBalance: %s\nResidents: %s\nTrusted/Outlaws: %s",
				sizeStr, balanceStr, residentsStr, trustedOutlawsStr,
			), true),
		},
	}

	status := town.Status
	AddField(embed, "Status", fmt.Sprintf(
		"%s Open\n%s Public\n%s Neutral\n%s Can Outsiders Spawn\n%s For Sale",
		BoolToEmoji(status.Open), BoolToEmoji(status.Public), BoolToEmoji(status.Neutral),
		BoolToEmoji(status.CanOutsidersSpawn), BoolToEmoji(status.ForSale),
	), true)

	if !status.Ruined {
		AddField(embed, "Overclaim Status", fmt.Sprintf("Overclaimed: `%s`\nShield: %s", town.OverclaimedString(), overclaimShield), true)
	}

	perms := town.Perms
	flags := perms.Flags

	AddField(embed, "Flags", fmt.Sprintf(
		"%s Explosions\n%s Mobs\n%s Fire\n%s PVP",
		BoolToEmoji(flags.Explosions), BoolToEmoji(flags.Mobs), BoolToEmoji(flags.Fire), BoolToEmoji(flags.PVP),
	), true)

	build, destroy, sw, itemUse := perms.GetPermStrings()
	AddField(embed, "Permissions", fmt.Sprintf(
		"Build: `%s`\nDestroy: `%s`\nSwitch: `%s`\nItem Use: `%s`",
		build, destroy, sw, itemUse,
	), true)

	return embed
}

func NewNationEmbed(nation oapi.NationInfo, allianceStore *store.Store[database.Alliance]) *discordgo.MessageEmbed {
	foundedTs := nation.Timestamps.Registered / 1000 // Seconds
	dateFounded := fmt.Sprintf("<t:%d:R>", foundedTs)

	stats := nation.Stats
	spawn := nation.Coordinates.Spawn

	board := nation.Board
	if board != "" {
		board = fmt.Sprintf("*%s*", board)
	}

	capitalName := nation.Capital.Name
	leaderName := nation.King.Name

	// TODO: add bonus to stats field
	//nationBonus :=

	open := fmt.Sprintf("%s Open", lo.Ternary(nation.Status.Open, ":green_circle:", ":red_circle:"))
	public := fmt.Sprintf("%s Public", lo.Ternary(nation.Status.Public, ":green_circle:", ":red_circle:"))
	neutral := fmt.Sprintf("%s Neutral", lo.Ternary(nation.Status.Neutral, ":green_circle:", ":red_circle:"))

	townNames := parallel.Map(nation.Towns, func(e oapi.Entity, _ int) string { return e.Name })
	slices.Sort(townNames)

	townsStr := utils.HumanizedSprintf("`%d`", stats.NumTowns)
	residentsStr := utils.HumanizedSprintf("`%d`", stats.NumResidents)
	balanceStr := utils.HumanizedSprintf("`%.0f` %s", stats.Balance, shared.EMOJIS.GOLD_INGOT)
	//bonusStr := utils.HumanizedSprintf("`%d`")
	alliesEnemiesStr := utils.HumanizedSprintf("`%d`/`%d`", stats.NumAllies, stats.NumEnemies)
	sizeStr := utils.HumanizedSprintf("`%d` %s (Worth: `%d` %s)",
		nation.Size(), shared.EMOJIS.CHUNK,
		nation.Worth(), shared.EMOJIS.GOLD_INGOT,
	)

	statsStr := fmt.Sprintf(
		"Size: %s\nBalance: %s\nResidents: %s\nTowns: %s\nAllies/Enemies: %s",
		sizeStr, balanceStr, residentsStr, townsStr, alliesEnemiesStr,
	)

	rankLines := []string{}
	for rank, players := range nation.Ranks {
		names := lo.Map(players, func(e oapi.Entity, _ int) string {
			return fmt.Sprintf("`%s`", e.Name)
		})

		line := fmt.Sprintf("[%d] %s", len(names), rank)
		if len(names) > 0 {
			line += fmt.Sprintf("\n%s", strings.Join(names, ", "))
		}

		rankLines = append(rankLines, line)
	}

	locationLink := fmt.Sprintf(
		"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=4)",
		spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
	)

	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Nation Information | `%s`", nation.Name),
		Description: board,
		Color:       nation.FillColourInt(),
		Footer:      DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Leader", fmt.Sprintf("[%s](%s)", leaderName, shared.NAMEMC_URL+nation.King.UUID), true),
			NewEmbedField("Capital", fmt.Sprintf("`%s`", capitalName), true),
			NewEmbedField("Location", locationLink, true),
			NewEmbedField("Stats", statsStr, true),
			NewEmbedField("Status", fmt.Sprintf("%s\n%s\n%s", open, public, neutral), true),
			NewEmbedField("Colours", fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", nation.MapColourFill, nation.MapColourOutline), true),
		},
	}

	ranksStr := strings.Join(rankLines, "\n\n")
	if len(ranksStr) < discordutil.EMBED_FIELD_VALUE_LIMIT {
		AddField(embed, "Ranks", ranksStr, false)
	}

	townListStr := strings.Join(townNames, ", ")
	if len(townListStr) <= discordutil.EMBED_FIELD_VALUE_LIMIT {
		townListStr = fmt.Sprintf("```%s```", townListStr)
		AddField(embed, fmt.Sprintf("Towns [%d]", stats.NumTowns), townListStr, false)
	}

	//#region Alliances field
	if allianceStore != nil {
		allianceByID := allianceStore.EntriesFunc(func(a database.Alliance) string {
			return a.Identifier
		})

		seen := sets.New[string]()
		allianceStore.ForEach(func(_ string, a database.Alliance) {
			if a.OwnNations.Has(nation.UUID) {
				seen.Append(a.Label)
				if a.Parent != nil {
					if parent, ok := allianceByID[*a.Parent]; ok {
						seen.Append(parent.Label)
					}
				}
			}
		})

		seenCount := len(seen)
		if seenCount > 0 {
			relatedAlliancesStr := fmt.Sprintf("```%s```", strings.Join(seen.Keys(), ", "))
			AddField(embed, fmt.Sprintf("Alliances [%d]", seenCount), relatedAlliancesStr, false)
		}
	}
	//#endregion

	AddField(embed, "Founded", dateFounded, true)

	if nation.Wiki != "" {
		AddField(embed, "Wiki", fmt.Sprintf("[Visit wiki page](%s)", nation.Wiki), true)
	}

	return embed
}

func NewTownlessPageEmbed(names []string) (*discordgo.MessageEmbed, error) {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("[%d] Townless Players", len(names)),
		Description: fmt.Sprintf("```%s```", strings.Join(names, "\n")),
	}

	return embed, nil
}

// Returns a circular emoji equivalent to the value of v
// where true becomes a green check, false a red cross.
func BoolToEmoji(v bool) string {
	if v {
		return shared.EMOJIS.CIRCLE_CHECK
	}

	return shared.EMOJIS.CIRCLE_CROSS
}
