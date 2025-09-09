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

var DEFAULT_FOOTER = &dgo.MessageEmbedFooter{
	IconURL: "https://cdn.discordapp.com/avatars/263377802647175170/a_0cd469f208f88cf98941123eb1b52259.webp?size=512&animated=true",
	Text:    "Maintained by Owen3H • Open Source on GitHub 💛", // unless you maintain your own fork, pls keep this as is :)
}

var EmbedField = discordutil.EmbedField

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
	var representativeValue string = "None"
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
			EmbedField("Leader(s)", leadersValue, false),
			EmbedField("Representative", representativeValue, true),
			EmbedField(nationsKey, nationsValue, false),
			EmbedField("Created At", fmt.Sprintf("<t:%d:f>", a.CreatedTimestamp()/1000), true),
		},
	}

	if a.UpdatedTimestamp != nil {
		updatedField := EmbedField("Last Updated", fmt.Sprintf("<t:%d:R>", *a.UpdatedTimestamp), true)
		embed.Fields = append(embed.Fields, updatedField)
	}

	return embed
}

func NewPlayerEmbed(resident oapi.PlayerInfo) *dgo.MessageEmbed {
	registeredTs := resident.Timestamps.Registered   // ms
	lastOnlineTs := resident.Timestamps.LastOnline   // ms
	joinedTownTs := resident.Timestamps.JoinedTownAt // ms

	status := "Offline" // Assume they are offline
	if resident.Status.IsOnline {
		status = "Online"
	} else {
		status = lo.Ternary(lastOnlineTs == nil, "Offline", fmt.Sprintf("Offline (Last Online: <t:%d:R>)", *lastOnlineTs/1000))
	}

	townName := lo.Ternary(resident.Town.Name == nil, "No Town", *resident.Town.Name)
	nationName := lo.Ternary(resident.Nation.Name == nil, "No Nation", *resident.Nation.Name)

	resName := resident.Name
	alias := resName

	// Add Prefix if available
	if resTitle := utils.CheckAlphanumeric(resident.Title); resTitle != "" {
		alias = fmt.Sprintf("%s %s", resTitle, resName)
	}

	// Add Postfix if available
	if resSurname := utils.CheckAlphanumeric(resident.Surname); resSurname != "" {
		alias = fmt.Sprintf("%s %s", resName, resSurname)
	}

	affiliation := lo.Ternary(townName == "No Town", "None (Townless)", fmt.Sprintf("%s (%s)", townName, nationName))
	affiliationField := EmbedField("Affiliation", affiliation, false)

	rank := "Resident"
	if resident.Status.IsMayor {
		rank = "Mayor"
	}
	if resident.Status.IsKing {
		rank = "Nation Leader"
	}

	embed := &dgo.MessageEmbed{
		Type:  dgo.EmbedTypeRich,
		Color: 7419530,
		Thumbnail: &dgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", resident.UUID),
		},
		Title: fmt.Sprintf("Player Information | `%s`", resName),
		Fields: []*dgo.MessageEmbedField{
			affiliationField,
			EmbedField("Rank", rank, true),
			EmbedField("Balance", utils.HumanizedSprintf("`%.0f`G %s", resident.Stats.Balance, EMOJIS.GOLD_INGOT), true),
			EmbedField("Status", status, true),
		},
	}

	if resident.About != nil {
		about := *resident.About
		if about != "" && about != DEFAULT_ABOUT {
			embed.Description = fmt.Sprintf("*%s*", about)
		}
	}

	// Alias differs from name (has surname and/or title)
	if alias != resName {
		field := EmbedField("Alias", alias, false)
		embed.Fields = append([]*dgo.MessageEmbedField{field}, embed.Fields...) // Insert at slice start
	}

	if joinedTownTs != nil {
		field := EmbedField("Joined Town", fmt.Sprintf("<t:%d:R>", *joinedTownTs/1000), false)
		embed.Fields = append(embed.Fields, field)
	}

	registeredField := EmbedField("Registered", fmt.Sprintf("<t:%d:R>", registeredTs/1000), false)
	embed.Fields = append(embed.Fields, registeredField)

	friendsStr := "No Friends :("
	if resident.Stats.NumFriends > 0 {
		friends := lop.Map(resident.Friends, func(e oapi.Entity, _ int) string {
			return e.Name
		})

		slices.Sort(friends)
		friendsStr = fmt.Sprintf("```%s```", strings.Join(friends, ", "))
	}

	friendsField := EmbedField("Friends", friendsStr, false)
	embed.Fields = append(embed.Fields, friendsField)

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
		Color:       utils.HexToInt("2ecc71"), // GREEN
		Fields: []*dgo.MessageEmbedField{
			EmbedField("Date Founded", fmt.Sprintf("<t:%d:R>", foundedTs), true),
			EmbedField("Founder", fmt.Sprintf("`%s`", town.Founder), true),
			EmbedField("Mayor", fmt.Sprintf("`%s`", town.Mayor.Name), true),
			EmbedField("Area", utils.HumanizedSprintf("`%d`/`%d` Chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
			EmbedField("Balance", utils.HumanizedSprintf("`%0.0f`G", town.Bal()), true),
			EmbedField("Residents", utils.HumanizedSprintf("`%d`", town.Stats.NumResidents), true),
			EmbedField("Overclaim Status", fmt.Sprintf("Overclaimed: `%s`\nShield: %s", town.OverclaimedString(), overclaimShield), false),
		},
	}
}

func NewNationEmbed(nation oapi.NationInfo) *dgo.MessageEmbed {
	foundedTs := nation.Timestamps.Registered / 1000 // Seconds
	dateFounded := fmt.Sprintf("<t:%d:R>", foundedTs)

	stats := nation.Stats
	spawn := nation.Coordinates.Spawn

	return &dgo.MessageEmbed{
		Type:  dgo.EmbedTypeRich,
		Title: fmt.Sprintf("Nation | %s", nation.Name),
		Color: nation.FillColourInt(),
		Fields: []*dgo.MessageEmbedField{
			EmbedField("King", nation.King.Name, true),
			EmbedField("Capital", nation.Capital.Name, true),
			EmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", spawn.X, spawn.Z, spawn.X, spawn.Y, spawn.Z), true),
			EmbedField("Date Founded", dateFounded, true),
			EmbedField("Area", fmt.Sprintf("%d Chunks", stats.NumTownBlocks), true),
			EmbedField("Balance", fmt.Sprintf("%.0fG", stats.Balance), true),
			EmbedField("Residents", fmt.Sprintf("`%d`", stats.NumResidents), false),
			EmbedField("Allies/Enemies", fmt.Sprintf("`%d` / `%d`", stats.NumAllies, stats.NumEnemies), false),
		},
	}
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
		Color:       15844367,
	}, nil
}
