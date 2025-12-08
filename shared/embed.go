package shared

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

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
func GetAffiliationLines(players map[string]oapi.PlayerInfo) string {
	var str string
	if players != nil {
		lines := []string{}
		for _, p := range players {
			line := fmt.Sprintf("`%s`", p.Name)
			if p.Town.Name != nil {
				line += fmt.Sprintf(" of %s", *p.Town.Name)
				if p.Nation.Name != nil {
					line += fmt.Sprintf(" (**%s**)", *p.Nation.Name)
				}
			} else {
				line += " (Townless)"
			}

			lines = append(lines, line)
		}

		str = strings.Join(lines, "\n")
	}

	return str
}

// Creates a single embed showing info from the given Alliance.
func NewAllianceEmbed(s *discordgo.Session, allianceStore *store.Store[database.Alliance], a database.Alliance) *discordgo.MessageEmbed {
	// Resort to dark blue unless alliance has optional fill colour specified.
	embedColour := discordutil.DARK_AQUA
	colours := a.Optional.Colours
	if colours != nil && colours.Fill != nil {
		embedColour = utils.HexToInt(*colours.Fill)
	}

	// Leader field logic
	leadersValue := "`None`"
	leaders, err := a.QueryLeaders() // TODO: Do we want to send an OAPI req for leaders every time?
	if err != nil {
		fmt.Printf("ERROR | Could not get leaders for alliance %s:\n%v", a.Identifier, err)
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
	nationStore, _ := database.GetStoreForMap(ACTIVE_MAP, database.NATIONS_STORE)
	nations, towns, residents, area, wealth := a.GetStats(nationStore, allianceStore)
	nationNames := lo.Map(nations, func(n oapi.NationInfo, _ int) string {
		return n.Name
	})

	nationsLen := len(nationNames)
	if nationsLen > 0 {
		slices.Sort(nationNames) // Alphabet sort
	}

	nationsKey := fmt.Sprintf("Nations [%d]", nationsLen)
	nationsValue := fmt.Sprintf("```%s```", strings.Join(nationNames, ", ")) // TODO: ALSO INCLUDE NATIONS FROM CHILD ALLIANCES

	registered := a.CreatedTimestamp() / 1000

	stats := fmt.Sprintf("Towns: %s\nResidents: %s\nSize: %s",
		utils.HumanizedSprintf("`%d`", len(towns)),
		utils.HumanizedSprintf("`%d`", residents),
		utils.HumanizedSprintf("`%d` %s (Worth `%d` %s)", area, EMOJIS.CHUNK, wealth, EMOJIS.GOLD_INGOT),
	)

	embed := &discordgo.MessageEmbed{
		Color:  embedColour,
		Footer: DEFAULT_FOOTER,
		Title:  fmt.Sprintf("Alliance Info | `%s`", a.Label),
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

	AddField(embed, nationsKey, nationsValue, false)
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
		parentAlliance, err := allianceStore.GetKey(strings.ToLower(*a.Parent))
		if err == nil {
			embed.Description = fmt.Sprintf("*This alliance is a puppet of `%s` / `%s`*.", parentAlliance.Identifier, parentAlliance.Label)
		}
	}

	if a.Optional.DiscordCode != nil {
		embed.Title = fmt.Sprintf("Alliance Info | %s", a.Label)
		embed.URL = fmt.Sprintf("https://discord.gg/%s", *a.Optional.DiscordCode)
	}

	return embed
}

func NewPlayerEmbed(player oapi.PlayerInfo) *discordgo.MessageEmbed {
	registeredTs := player.Timestamps.Registered   // ms
	lastOnlineTs := player.Timestamps.LastOnline   // ms
	joinedTownTs := player.Timestamps.JoinedTownAt // ms

	status := "Offline" // Assume they are offline
	if player.Status.IsOnline {
		status = "Online"
	} else {
		status = lo.Ternary(lastOnlineTs == nil, "Offline", fmt.Sprintf("Offline (Last Online: <t:%d:R>)", *lastOnlineTs/1000))
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
		mdb, _ := database.Get(ACTIVE_MAP)
		townsStore, _ := database.GetStore(mdb, database.TOWNS_STORE)
		town, err := townsStore.GetKey(*player.Town.UUID)

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
		friends := lop.Map(player.Friends, func(e oapi.Entity, _ int) string {
			return e.Name
		})

		slices.Sort(friends)
		friendsStr = fmt.Sprintf("```%s```", strings.Join(friends, ", "))
	}

	title := fmt.Sprintf("Player Information | `%s`", playerName)

	// Alias differs from name (has surname and/or title)
	if alias != playerName {
		title += fmt.Sprintf(" aka \"%s\"", alias)
	}

	townRanks := player.Ranks.Town
	nationRanks := player.Ranks.Nation

	townRanksStr := "No ranks"
	if len(townRanks) > 0 {
		townRanksStr = strings.Join(lo.Map(townRanks, func(r string, _ int) string {
			return fmt.Sprintf("`%s`", r)
		}), ", ")
	}

	nationRanksStr := "No ranks"
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
			NewEmbedField("Balance", utils.HumanizedSprintf("`%.0f`G %s", player.Stats.Balance, EMOJIS.GOLD_INGOT), true),
			NewEmbedField("Status", status, true),
		},
	}

	if player.About != nil {
		about := *player.About
		if about != "" && about != DEFAULT_ABOUT {
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
	if town.Board != DEFAULT_TOWN_BOARD {
		desc = fmt.Sprintf("*%s*", town.Board)
	}

	overclaimShield := "`Inactive` " + EMOJIS.SHIELD_RED
	if town.Status.HasOverclaimShield {
		overclaimShield = "`Active` " + EMOJIS.SHIELD_GREEN
	}

	nationName := "No Nation"
	nationJoin := ""
	if town.Nation.Name != nil {
		nationName = *town.Nation.Name
		nationJoin = fmt.Sprintf(" (Joined <t:%d:R>)", *town.Timestamps.JoinedNationAt/1000)
	}

	locX := town.Coordinates.Spawn.X
	locY := town.Coordinates.Spawn.Y
	locZ := town.Coordinates.Spawn.Z

	sizeStr := utils.HumanizedSprintf("`%d`/`%d` %s (Worth: `%d` %s)", town.Size(), town.MaxSize(), EMOJIS.CHUNK, town.Worth(), EMOJIS.GOLD_INGOT)
	balanceStr := utils.HumanizedSprintf("`%.0f`G %s", town.Bal(), EMOJIS.GOLD_INGOT)
	residentsStr := utils.HumanizedSprintf("`%d`", town.Stats.NumResidents)
	outlawsStr := utils.HumanizedSprintf("`%d`", town.Stats.NumOutlaws)
	trustedStr := utils.HumanizedSprintf("`%d`", town.Stats.NumTrusted)

	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       townTitle,
		Description: desc,
		Color:       colour,
		Footer:      DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			//NewEmbedField("Date Founded", fmt.Sprintf("<t:%d:R>", foundedTs), true),
			NewEmbedField("Origin", fmt.Sprintf("Founded <t:%d:R> by `%s`", foundedTs, town.Founder), false),
			NewEmbedField("Mayor", fmt.Sprintf("`%s`", town.Mayor.Name), true),
			NewEmbedField("Nation", fmt.Sprintf("`%s`%s", nationName, nationJoin), true),
			NewEmbedField("Location (XYZ)", fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=3)", locX, locY, locZ, locX, locZ), true),
			NewEmbedField("Stats", fmt.Sprintf(
				"Size: %s\nBalance: %s\nResidents: %s\nTrusted: %s\nOutlaws: %s",
				sizeStr, balanceStr, residentsStr, trustedStr, outlawsStr,
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

func NewNationEmbed(nation oapi.NationInfo) *discordgo.MessageEmbed {
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

	townNames := lop.Map(nation.Towns, func(e oapi.Entity, _ int) string {
		return e.Name
	})
	slices.Sort(townNames)

	townsStr := strings.Join(townNames, ", ")
	if len(townsStr) > discordutil.EMBED_FIELD_VALUE_LIMIT {
		townsStr = "Too many towns to display!\nClick the **View All Towns** button to see the full list."
	} else {
		townsStr = fmt.Sprintf("```%s```", townsStr)
	}

	sizeStr := utils.HumanizedSprintf("`%d` %s (Worth: `%d` %s)", nation.Size(), EMOJIS.CHUNK, nation.Worth(), EMOJIS.GOLD_INGOT)
	residentsStr := utils.HumanizedSprintf("`%d`", stats.NumResidents)
	balanceStr := utils.HumanizedSprintf("`%.0f`G%s  ", stats.Balance, EMOJIS.GOLD_INGOT)
	//bonusStr := utils.HumanizedSprintf("`%d`")
	alliesEnemiesStr := utils.HumanizedSprintf("`%d`/`%d`", stats.NumAllies, stats.NumEnemies)

	statsStr := fmt.Sprintf(
		"Size: %s\nBalance: %s\nResidents: %s\nAllies/Enemies: %s",
		sizeStr, balanceStr, residentsStr, alliesEnemiesStr,
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

	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Nation Information | `%s`", nation.Name),
		Description: board,
		Color:       nation.FillColourInt(),
		Footer:      DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Leader", fmt.Sprintf("[%s](%s)", leaderName, NAMEMC_URL+nation.King.UUID), true),
			NewEmbedField("Capital", fmt.Sprintf("`%s`", capitalName), true),
			NewEmbedField("Location (XZ)", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", spawn.X, spawn.Z, spawn.X, spawn.Y, spawn.Z), true),
			NewEmbedField("Stats", statsStr, true),
			NewEmbedField("Status", fmt.Sprintf("%s\n%s\n%s", open, public, neutral), true),
			NewEmbedField("Colours", fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", nation.MapColourFill, nation.MapColourOutline), true),
		},
	}

	ranksStr := strings.Join(rankLines, "\n\n")
	if len(ranksStr) < discordutil.EMBED_FIELD_VALUE_LIMIT {
		AddField(embed, "Ranks", ranksStr, false)
	}

	AddField(embed, fmt.Sprintf("Towns [%d]", stats.NumTowns), townsStr, false)
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

func NewStaffEmbed() (*discordgo.MessageEmbed, error) {
	var onlineStaff []string
	var errors []error

	ids := []string{} // Fetch them from somewhere
	players, err := oapi.QueryPlayers(ids...)

	// Calls specified func for every slice element in parallel.
	lop.ForEach(players, func(p oapi.PlayerInfo, _ int) {
		if err != nil {
			fmt.Println(err)
			errors = append(errors, err)

			return
		}

		if p.Status.IsOnline {
			onlineStaff = append(onlineStaff, p.Name)
		}
	})

	if len(errors) > 0 {
		return nil, errors[0]
	}

	slices.Sort(onlineStaff)

	content := "None"
	if len(onlineStaff) > 0 {
		content = strings.Join(onlineStaff, ", ")
	}

	return &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       "Staff List | Online",
		Description: fmt.Sprintf("```%s```", content),
		Color:       discordutil.GOLD,
	}, nil
}

func BoolToEmoji(v bool) string {
	if v {
		return EMOJIS.CIRCLE_CHECK
	}

	return EMOJIS.CIRCLE_CROSS
}
