package shared

import (
	"emcsrw/internal/database"
	"emcsrw/internal/database/store"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"emcsrw/pkg/utils/logutil"
	"emcsrw/pkg/utils/sets"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/samber/lo/parallel"

	"github.com/bwmarrin/discordgo"
)

var npcRgx = regexp.MustCompile(`^NPC\d+$`)

// Creates a single embed showing info from the given Alliance.
func NewAllianceEmbed(
	s *discordgo.Session, mdb *database.Database,
	a database.Alliance, rankInfo *database.AllianceRankInfo,
) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	playerStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
	if err != nil {
		logutil.Printf(logutil.RED, "ERR | Could not get player store for map %s:\n%v", mdb.Name(), err)
		return nil, nil
	}

	// Leader field logic
	leadersValue := "`None`"
	leaders, err := a.Leaders(playerStore)
	if err == nil {
		leadersValue = BuildAffiliationsString(leaders)
	}
	// } else {
	// 	logutil.Printf(logutil.YELLOW, "ERR | Could not get leaders for alliance %s:\n\t%v", a.Identifier, err)
	// }

	// Representative field logic
	representativeValue := "`None`"
	if a.RepresentativeID != nil {
		u, err := s.User(*a.RepresentativeID)
		if err == nil {
			representativeValue = fmt.Sprintf("`%s`", u.String())
		}
	}

	// Nation field logic
	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		logutil.Printf(logutil.RED, "ERR | Could not get nation store for map %s:\n%v", mdb.Name(), err)
		return nil, nil
	}

	ownNations := nationStore.GetFromSet(a.OwnNations)
	ownNationNames := lo.Map(ownNations, func(n oapi.NationInfo, _ int) string { return n.Name })
	if len(ownNationNames) > 0 {
		slices.Sort(ownNationNames) // Alphabet sort
	}

	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		logutil.Printf(logutil.RED, "ERR | Could not get alliance store for map %s:\n%v", mdb.Name(), err)
		return nil, nil
	}

	alliances := allianceStore.Values()
	childAlliances := a.ChildAlliances(alliances)
	childNations := nationStore.GetFromSet(childAlliances.NationIds())

	nationsAmt := len(ownNations) + len(childNations)
	towns, residentsAmt, area, wealth := a.Stats(ownNations, childNations)
	stats := fmt.Sprintf("Towns: %s\nNations: %s\nResidents: %s\nSize: %s",
		logutil.HumanizedSprintf("`%d`", len(towns)),
		logutil.HumanizedSprintf("`%d`", nationsAmt),
		logutil.HumanizedSprintf("`%d` %s", residentsAmt, EMOJIS.RESIDENT_PURPLE),
		logutil.HumanizedSprintf("`%d` %s (Worth `%d` %s)", area, EMOJIS.CHUNK, wealth, EMOJIS.GOLD_INGOT),
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

	desc := fmt.Sprintf("**Leaders(s)**\n%s\n\n**Discord Representative**\n%s", leadersValue, representativeValue)

	embed := discordutil.NewEmbedBuilder(&embedColour, &title, &desc, nil)
	embed.AddField("Stats", stats, true)

	coloursStr := "No colours set."
	if a.Optional.Colours != nil && a.Optional.Colours.Fill != nil {
		fill := *a.Optional.Colours.Fill

		outline := fill
		if a.Optional.Colours.Outline != nil {
			outline = *a.Optional.Colours.Outline
		}

		coloursStr = fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", fill, outline)
	}

	embed.AddField("Colours", coloursStr, true)
	embed.AddField("Type", fmt.Sprintf("`%s`", a.Type.Colloquial()), true)
	embed.AddField("Registered", fmt.Sprintf("<t:%d:f>\n<t:%d:R>", registered, registered), true)

	if a.UpdatedTimestamp != nil {
		updatedSec := *a.UpdatedTimestamp / 1000
		embed.AddField("Last Updated", fmt.Sprintf("<t:%d:f>\n<t:%d:R>", updatedSec, updatedSec), true)
	}

	ownNationsValue := fmt.Sprintf("```%s```", strings.Join(ownNationNames, ", "))
	if len(childNations) < 1 {
		if len(ownNationsValue) > discordutil.EMBED_FIELD_VALUE_LIMIT {
			ownNationsValue = "Too many nations to display. Use `/alliance nations` to see the full list."
		}

		embed.AddField(fmt.Sprintf("Nations [%d]", len(ownNations)), ownNationsValue, false)
	} else {
		if len(ownNationsValue) > discordutil.EMBED_FIELD_VALUE_LIMIT {
			ownNationsValue = "Too many nations to display. Use `/alliance nations` to see the full list."
		}

		embed.AddField(fmt.Sprintf("Self Nations [%d]", len(ownNations)), ownNationsValue, false)

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

		if len(childNationsValue) > discordutil.EMBED_FIELD_VALUE_LIMIT {
			childNationsValue = "Too many puppet nations to display. Use `/alliance nations` to see the full list."
		}

		embed.AddField(childNationsKey, childNationsValue, false)
	}

	newsStore, err := database.GetStoreForMap(ACTIVE_MAP, database.NEWS_STORE)
	if err == nil {
		allianceNews := database.GetAllianceNews(newsStore, a)
		if len(allianceNews) > 0 {
			recentNewsStr, count := BuildNewsString(allianceNews, 2, discordutil.EMBED_FIELD_VALUE_LIMIT)
			embed.AddField(fmt.Sprintf("Recent News [%d]", count), recentNewsStr, false)
		}
	}

	if flag := a.Optional.ImageURL; flag != nil {
		if *flag != "" {
			embed.SetThumbnail(*flag, nil)
		}
	}

	if a.Parent != nil {
		parentAlliance, err := allianceStore.Get(strings.ToLower(*a.Parent))
		if err == nil {
			existingDesc := embed.Description
			puppetStr := fmt.Sprintf("*This alliance is a puppet of `%s` / `%s`*.", parentAlliance.Identifier, parentAlliance.Label)
			embed.SetDescription(fmt.Sprintf("%s\n\n%s", puppetStr, existingDesc))
		}
	}

	// TODO: Return the MessageBuilder itself or make a seperate component builder.
	// 		 Building a whole message just for the components is confusing.
	b := discordutil.NewMessageBuilder()
	if a.Optional.DiscordCode != nil {
		inviteURL := fmt.Sprintf("https://discord.gg/%s", *a.Optional.DiscordCode)
		b.AddButton("Join discord", discordgo.LinkButton, &inviteURL, &discordutil.DISCORD_EMOJI, nil)
	}

	return embed.Build(), b.BuildComponents()
}

// Builds an embed that describes a user with only minimal info.
//
// Should be preferred when the user is annoying and has opted-out of the Official API.
func NewBasicPlayerEmbed(player database.BasicPlayer, description string) *discordgo.MessageEmbed {
	var town *oapi.TownInfo
	if player.Town != nil {
		townsStore, _ := database.GetStoreForMap(ACTIVE_MAP, database.TOWNS_STORE)
		town, _ = townsStore.Get(player.Town.UUID)
	}

	townName := lo.TernaryF(town == nil, func() string { return "No Town" }, func() string { return town.Name })
	nationName := lo.TernaryF(
		town == nil || town.Nation.Name == nil,
		func() string { return "No Nation" },
		func() string { return *town.Nation.Name },
	)

	affiliation := "None (Townless)"
	if town != nil {
		spawn := town.Coordinates.Spawn
		affiliation = fmt.Sprintf("[%s](https://map.earthmc.net?x=%f&z=%f&zoom=3) (%s)", townName, spawn.X, spawn.Z, nationName)
	}

	title := fmt.Sprintf("Player Information | `%s`", player.Name)
	embed := discordutil.NewEmbedBuilder(&discordutil.DARK_PURPLE, &title, &description, nil)
	embed.SetFields(
		NewEmbedField("Affiliation", affiliation, true),
		NewEmbedField("Rank", player.RankString(), true), // TODO: Add town ranks using player.Town.Ranks
		NewEmbedField("Minecraft UUID", fmt.Sprintf("`%s`", player.UUID), false),
	)

	sb, err := NewSkinBuilder(player.UUID, SkinTypeBust3D, SkinFormatPNG, 230)
	if err == nil {
		embed.SetThumbnail(sb.Build(), nil)
	}

	return embed.Build()
}

// Builds an embed that describes a user using info from the Official API.
//
// Should be preferred when the user exists on said API and has not opted-out.
func NewPlayerEmbed(s *discordgo.Session, player oapi.PlayerInfo) *discordgo.MessageEmbed {
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
		townsStore, _ := database.GetStoreForMap(ACTIVE_MAP, database.TOWNS_STORE)
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

	townRanks, townRanksStr := player.Ranks.Town, "No ranks"
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

	embed := discordutil.NewEmbedBuilder(&discordutil.DARK_PURPLE, &title, nil, nil)
	embed.SetFields(
		// Affiliation (prepended)
		// Rank (prepended)
		NewEmbedField("Balance", logutil.HumanizedSprintf("`%.0f` %s", player.Stats.Balance, EMOJIS.GOLD_INGOT), true),
		NewEmbedField("Status", status, true),
	)

	sb, err := NewSkinBuilder(player.UUID, SkinTypeBust3D, SkinFormatPNG, 230)
	if err == nil {
		embed.SetThumbnail(sb.Build(), nil)
	}

	if player.About != nil {
		about := *player.About
		if about != "" && about != DEFAULT_ABOUT {
			embed.SetDescription(fmt.Sprintf("*%s*", about))
		}
	}

	if joinedTownTs != nil {
		embed.AddField("Joined Town", fmt.Sprintf("<t:%d:R>", *joinedTownTs/1000), true)
	}

	embed.AddField("Registered", fmt.Sprintf("<t:%d:R>", registeredTs/1000), true)
	embed.AddField("Appointed Ranks", fmt.Sprintf("Town: %s\nNation: %s", townRanksStr, nationRanksStr), false)
	embed.AddField("Friends", friendsStr, false)
	embed.AddField("Minecraft UUID", fmt.Sprintf("`%s`", player.UUID), false)

	if player.Discord != nil {
		mentionStr := fmt.Sprintf("<@%s>", *player.Discord)
		if user, err := s.User(*player.Discord); err != nil {
			mentionStr = fmt.Sprintf("<@%s> (%s)", user.ID, user.String())
		}

		embed.AddField("Discord", mentionStr, false)
	}

	// Second field
	if townName != "No Town" {
		embed.PrependField("Rank", rank, true)
	}

	// First field
	embed.PrependField("Affiliation", affiliation, true)

	return embed.Build()
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
	if town.Board != DEFAULT_TOWN_BOARD {
		desc = fmt.Sprintf("*%s*", town.Board)
	}

	nationName := "No Nation"
	nationJoin := ""
	if town.Nation.Name != nil {
		nationName = *town.Nation.Name
		nationJoin = fmt.Sprintf(" (Joined <t:%d:R>)", *town.Timestamps.JoinedNationAt/1000)
	}

	spawn := town.Coordinates.Spawn

	balanceStr := logutil.HumanizedSprintf("`%.0f` %s", town.Bal(), EMOJIS.GOLD_INGOT)
	residentsStr := logutil.HumanizedSprintf("`%d` %s", town.Stats.NumResidents, EMOJIS.RESIDENT_PURPLE)
	trustedOutlawsStr := logutil.HumanizedSprintf("`%d`/`%d`", town.Stats.NumTrusted, town.Stats.NumOutlaws)

	sizeStr := logutil.HumanizedSprintf("`%d`/`%d` %s (Worth: `%d` %s)",
		town.Size(), town.MaxSize(),
		EMOJIS.CHUNK, town.Worth(), EMOJIS.GOLD_INGOT,
	)

	locationLink := fmt.Sprintf(
		"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)",
		spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
	)

	mayorStr := fmt.Sprintf("`%s`", town.Mayor.Name)
	if npcRgx.MatchString(town.Mayor.Name) {
		mayorStr = fmt.Sprintf("[%s](%s)", town.Mayor.Name, NAMEMC_URL+town.Mayor.UUID)
	}

	founderStr := fmt.Sprintf("`%s`", town.Founder)
	if npcRgx.MatchString(town.Founder) {
		founderStr = fmt.Sprintf("[%s](%s)", town.Founder, NAMEMC_URL+town.Founder)
	}

	status := town.Status
	flags := town.Perms.Flags
	build, destroy, sw, itemUse := town.Perms.EncodeAll()

	embed := discordutil.NewEmbedBuilder(&colour, &townTitle, &desc, nil)
	embed.SetFields(
		NewEmbedField("Origin", fmt.Sprintf("Founded <t:%d:R> by %s", foundedTs, founderStr), false),
		NewEmbedField("Mayor", mayorStr, true),
		NewEmbedField("Nation", fmt.Sprintf("`%s`%s", nationName, nationJoin), true),
		NewEmbedField("Location", locationLink, true),
		NewEmbedField("Stats", fmt.Sprintf(
			"Size: %s\nBalance: %s\nResidents: %s\nTrusted/Outlaws: %s",
			sizeStr, balanceStr, residentsStr, trustedOutlawsStr,
		), false),
		NewEmbedField("Status", fmt.Sprintf(
			"%s Open\n%s Public\n%s Neutral\n%s Can Outsiders Spawn\n%s Overclaimed\n%s For Sale",
			BoolToEmoji(status.Open), BoolToEmoji(status.Public), BoolToEmoji(status.Neutral),
			BoolToEmoji(status.CanOutsidersSpawn), BoolToEmoji(status.Overclaimed), BoolToEmoji(status.ForSale),
		), true),
		NewEmbedField("Flags", fmt.Sprintf(
			"%s Explosions\n%s Mobs\n%s Fire\n%s PVP",
			BoolToEmoji(flags.Explosions), BoolToEmoji(flags.Mobs), BoolToEmoji(flags.Fire), BoolToEmoji(flags.PVP),
		), true),
		NewEmbedField("Permissions", fmt.Sprintf(
			"Build: `%s`\nDestroy: `%s`\nSwitch: `%s`\nItem Use: `%s`",
			build, destroy, sw, itemUse,
		), true),
	)

	return embed.Build()
}

func NewNationEmbed(
	nation oapi.NationInfo,
	newsStore *store.Store[database.NewsEntry],
	allianceStore *store.Store[database.Alliance],
) *discordgo.MessageEmbed {
	board := nation.Board
	if board != "" {
		board = fmt.Sprintf("*%s*", board)
	}

	capitalName := nation.Capital.Name
	leaderName := nation.King.Name

	stats := nation.Stats
	spawn := nation.Coordinates.Spawn

	locationLink := fmt.Sprintf(
		"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=4)",
		spawn.X, spawn.Y, spawn.Z, spawn.X, spawn.Z,
	)

	townsResidentsStr := logutil.HumanizedSprintf("`%d`/`%d`", stats.NumTowns, stats.NumResidents)
	balanceStr := logutil.HumanizedSprintf("`%.0f` %s", stats.Balance, EMOJIS.GOLD_INGOT)
	bonusStr := logutil.HumanizedSprintf("`%d` %s", stats.NationBonus, EMOJIS.CHUNK)
	alliesEnemiesStr := logutil.HumanizedSprintf("`%d`/`%d`", stats.NumAllies, stats.NumEnemies)
	sizeStr := logutil.HumanizedSprintf("`%d` %s (Worth: `%d` %s)",
		nation.Size(), EMOJIS.CHUNK,
		nation.Worth(), EMOJIS.GOLD_INGOT,
	)

	statsStr := fmt.Sprintf(
		"Size: %s\nBalance: %s\nTowns/Residents: %s\nAllies/Enemies: %s\nClaim Bonus: %s",
		sizeStr, balanceStr, townsResidentsStr, alliesEnemiesStr, bonusStr,
	)

	open := fmt.Sprintf("%s Open", lo.Ternary(nation.Status.Open, ":green_circle:", ":red_circle:"))
	public := fmt.Sprintf("%s Public", lo.Ternary(nation.Status.Public, ":green_circle:", ":red_circle:"))
	neutral := fmt.Sprintf("%s Neutral", lo.Ternary(nation.Status.Neutral, ":green_circle:", ":red_circle:"))

	foundedTs := nation.Timestamps.Registered / 1000 // Seconds
	dateFounded := fmt.Sprintf("<t:%d:R>", foundedTs)

	leaderStr := fmt.Sprintf("`%s`", leaderName)
	if npcRgx.MatchString(leaderName) {
		leaderStr = fmt.Sprintf("[%s](%s)", leaderName, nation.King.UUID)
	}

	colour := nation.FillColourInt()
	title := fmt.Sprintf("Nation Information | `%s` | %s `%s`", nation.Name, "⭐", capitalName)
	embed := discordutil.NewEmbedBuilder(&colour, &title, &board, nil)
	embed.SetFields(
		NewEmbedField("Leader", leaderStr, true),
		NewEmbedField("Location", locationLink, true),
		NewEmbedField("Founded", dateFounded, true),
		NewEmbedField("Stats", statsStr, true),
		NewEmbedField("Status", fmt.Sprintf("%s\n%s\n%s", open, public, neutral), true),
		NewEmbedField("Colours", fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", nation.MapColourFill, nation.MapColourOutline), true),
	)

	ranksStr := BuildNationRanksString(nation)
	if len(ranksStr) < discordutil.EMBED_FIELD_VALUE_LIMIT {
		embed.AddField("Ranks", ranksStr, true)
	}

	townNames := parallel.Map(nation.Towns, func(e oapi.Entity, _ int) string { return e.Name })
	slices.Sort(townNames)

	townListStr := strings.Join(townNames, ", ")
	if len(townListStr) <= discordutil.EMBED_FIELD_VALUE_LIMIT {
		townListStr = fmt.Sprintf("```%s```", townListStr)
		embed.AddField(fmt.Sprintf("Towns [%d]", stats.NumTowns), townListStr, false)
	}

	//#region Alliances field
	if allianceStore != nil {
		allianceByID := allianceStore.EntriesFunc(func(a database.Alliance) string {
			return a.Identifier
		})

		seen := sets.New[string]()
		allianceStore.ForEach(func(_ string, a database.Alliance) {
			if !a.OwnNations.Has(nation.UUID) {
				return
			}

			seen.Add(a.Identifier)
			if a.Parent != nil {
				if parent, ok := allianceByID[*a.Parent]; ok {
					seen.Add(parent.Identifier)
				}
			}
		})

		seenCount := len(seen)
		if seenCount > 0 {
			relatedAlliancesStr := fmt.Sprintf("```%s```", strings.Join(seen.Keys(), ", "))
			embed.AddField(fmt.Sprintf("Alliances [%d]", seenCount), relatedAlliancesStr, false)
		}
	}
	//#endregion

	//#region Try add "Recent News" field
	if newsStore != nil {
		nationNews := database.GetNationNews(newsStore, nation)
		if len(nationNews) > 0 {
			recentNewsStr, count := BuildNewsString(nationNews, 2, discordutil.EMBED_FIELD_VALUE_LIMIT)
			embed.AddField(fmt.Sprintf("Recent News [%d]", count), recentNewsStr, false)
		}
	}
	//#endregion

	return embed.Build()
}

// func NewTownlessPageEmbed(names []string) (*discordgo.MessageEmbed, error) {
// 	title := fmt.Sprintf("[%d] Townless Players", len(names))
// 	desc := fmt.Sprintf("```%s```", strings.Join(names, "\n"))
// 	return discordutil.NewEmbedBuilder(&discordutil.PURPLE, &title, &desc, nil).Build(), nil
// }
