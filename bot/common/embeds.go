package common

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/database"
	"emcsrw/bot/discordutil"
	"emcsrw/utils"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	dgo "github.com/bwmarrin/discordgo"
)

var NewEmbedField = discordutil.NewEmbedField
var PrependField = discordutil.PrependField
var AddField = discordutil.AddField

var DEFAULT_FOOTER = &dgo.MessageEmbedFooter{
	IconURL: "https://cdn.discordapp.com/avatars/263377802647175170/a_0cd469f208f88cf98941123eb1b52259.webp?size=512&animated=true",
	Text:    "Maintained by Owen3H • Open Source on GitHub 💛", // unless you maintain your own fork, pls keep this as is :)
}

// Creates a single embed given alliance data. This is the output from `/alliance lookup`.
func NewAllianceEmbed(s *dgo.Session, a *database.Alliance) *dgo.MessageEmbed {
	// Resort to dark blue unless alliance has optional fill colour specified.
	embedColour := discordutil.DARK_AQUA
	colours := a.Optional.Colours
	if colours != nil && colours.Fill != nil {
		embedColour = utils.HexToInt(*colours.Fill)
	}

	// Leader field logic
	leadersValue := "None"
	leaders := a.Optional.Leaders
	if leaders != nil {
		leadersValue = strings.Join(*leaders, "\n")
	}

	// Representative field logic
	representativeValue := "None"
	if a.RepresentativeID != nil {
		u, err := s.User(strconv.FormatUint(*a.RepresentativeID, 10))
		if err != nil {
			representativeValue = u.Mention()
		}
	}

	// Nation field logic
	nationsLen := len(a.OwnNations)
	if nationsLen > 0 {
		slices.Sort(a.OwnNations) // Alphabet sort
	}

	nationsKey := fmt.Sprintf("Nations [%d]", nationsLen)
	nationsValue := fmt.Sprintf("```%s```", strings.Join(a.OwnNations, ", "))

	embed := &dgo.MessageEmbed{
		Color:  embedColour,
		Footer: DEFAULT_FOOTER,
		Title:  fmt.Sprintf("Alliance Info | `%s` (%s)", a.Label, a.Identifier),
		Fields: []*dgo.MessageEmbedField{
			NewEmbedField("Leader(s)", leadersValue, false),
			NewEmbedField("Representative", representativeValue, true),
			NewEmbedField(nationsKey, nationsValue, false),
			NewEmbedField("Created At", fmt.Sprintf("<t:%d:f>", a.CreatedTimestamp()/1000), true),
		},
	}

	if a.UpdatedTimestamp != nil {
		AddField(embed, "Last Updated", fmt.Sprintf("<t:%d:R>", *a.UpdatedTimestamp), true)
	}

	return embed
}

func NewPlayerEmbed(player oapi.PlayerInfo) *dgo.MessageEmbed {
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

	affiliation := lo.Ternary(townName == "No Town", "None (Townless)", fmt.Sprintf("%s (%s)", townName, nationName))

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

	embed := &dgo.MessageEmbed{
		Type:   dgo.EmbedTypeRich,
		Color:  discordutil.DARK_PURPLE,
		Footer: DEFAULT_FOOTER,
		Thumbnail: &dgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", player.UUID),
		},
		Title: title,
		Fields: []*dgo.MessageEmbedField{
			// affiliation (prepended)
			// rank (prepended)
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

func NewTownEmbed(town oapi.TownInfo) *dgo.MessageEmbed {
	foundedTs := town.Timestamps.Registered / 1000 // Seconds

	townTitle := fmt.Sprintf("Town Information | %s", town.Name)
	if town.Nation.Name != nil {
		townTitle += fmt.Sprintf(" (%s)", *town.Nation.Name)
	}

	desc := ""
	if town.Board != DEFAULT_TOWN_BOARD {
		desc = fmt.Sprintf("*%s*", town.Board)
	}

	overclaimShield := "`Inactive` " + EMOJIS.SHIELD_RED
	if town.Status.HasOverclaimShield {
		overclaimShield = "`Active` " + EMOJIS.SHIELD_GREEN
	}

	return &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       townTitle,
		Description: desc,
		Color:       discordutil.GREEN,
		Fields: []*dgo.MessageEmbedField{
			NewEmbedField("Date Founded", fmt.Sprintf("<t:%d:R>", foundedTs), true),
			NewEmbedField("Founder", fmt.Sprintf("`%s`", town.Founder), true),
			NewEmbedField("Mayor", fmt.Sprintf("`%s`", town.Mayor.Name), true),
			NewEmbedField("Area", utils.HumanizedSprintf("`%d`/`%d` Chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
			NewEmbedField("Balance", utils.HumanizedSprintf("`%0.0f`G", town.Bal()), true),
			NewEmbedField("Residents", utils.HumanizedSprintf("`%d`", town.Stats.NumResidents), true),
			NewEmbedField("Overclaim Status", fmt.Sprintf("Overclaimed: `%s`\nShield: %s", town.OverclaimedString(), overclaimShield), false),
		},
	}
}

func NewNationEmbed(nation oapi.NationInfo) *dgo.MessageEmbed {
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

	open := fmt.Sprintf("%s Open", lo.Ternary(nation.Status.Open, ":green_circle:", ":red_circle:"))
	public := fmt.Sprintf("%s Public", lo.Ternary(nation.Status.Public, ":green_circle:", ":red_circle:"))
	neutral := fmt.Sprintf("%s Neutral", lo.Ternary(nation.Status.Neutral, ":green_circle:", ":red_circle:"))

	towns := lop.Map(nation.Towns, func(e oapi.Entity, _ int) string {
		return e.Name
	})

	slices.Sort(towns)

	embed := &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Nation Information | `%s`", nation.Name),
		Description: board,
		Color:       nation.FillColourInt(),
		Fields: []*dgo.MessageEmbedField{
			NewEmbedField("Leader", fmt.Sprintf("[%s](%s)", leaderName, NAMEMC_URL+nation.King.UUID), true),
			NewEmbedField("Capital", fmt.Sprintf("`%s`", capitalName), true),
			NewEmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", spawn.X, spawn.Z, spawn.X, spawn.Y, spawn.Z), true),
			NewEmbedField("Size", utils.HumanizedSprintf("%s `%d` Chunks", EMOJIS.CHUNK, stats.NumTownBlocks), true),
			NewEmbedField("Residents", utils.HumanizedSprintf("`%d`", stats.NumResidents), true),
			NewEmbedField("Balance", utils.HumanizedSprintf("%s `%.0f`G", EMOJIS.GOLD_INGOT, stats.Balance), true),
			NewEmbedField("Allies/Enemies", fmt.Sprintf("`%d`/`%d`", stats.NumAllies, stats.NumEnemies), true),
			NewEmbedField("Status", fmt.Sprintf("%s\n%s\n%s", open, public, neutral), true),
			NewEmbedField("Colours", fmt.Sprintf("Fill: `#%s`\nOutline: `#%s`", nation.MapColourFill, nation.MapColourOutline), true),
			NewEmbedField(fmt.Sprintf("Towns [%d]", stats.NumTowns), fmt.Sprintf("```%s```", strings.Join(towns, ", ")), false),
			NewEmbedField("Founded", dateFounded, true),
		},
	}

	if nation.Wiki != nil {
		AddField(embed, "Wiki", fmt.Sprintf("[Visit wiki page](%s)", *nation.Wiki), true)
	}

	return embed
}

func NewTownlessPageEmbed(names []string) (*dgo.MessageEmbed, error) {
	embed := &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       fmt.Sprintf("[%d] Townless Players", len(names)),
		Description: fmt.Sprintf("```%s```", strings.Join(names, "\n")),
	}

	return embed, nil
}

func NewStaffEmbed() (*dgo.MessageEmbed, error) {
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

	return &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       "Staff List | Online",
		Description: fmt.Sprintf("```%s```", content),
		Color:       discordutil.GOLD,
	}, nil
}
