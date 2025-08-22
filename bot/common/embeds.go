package common

import (
	"emcsrw/oapi"
	"emcsrw/oapi/objs"
	"emcsrw/utils"
	"fmt"
	"strings"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	dgo "github.com/bwmarrin/discordgo"
)

func EmbedField(name string, value string, inline bool) *dgo.MessageEmbedField {
	return &dgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

func CreatePlayerEmbed(resident objs.PlayerInfo) *dgo.MessageEmbed {
	registeredTs := resident.Timestamps.Registered / 1000  // Seconds
	lastOnlineTs := *resident.Timestamps.LastOnline / 1000 // Seconds

	status := "Offline"
	if resident.Status.IsOnline {
		status = "Online"
	}

	townName := "No Town"
	if resident.Town.Name != nil {
		townName = *resident.Town.Name
	}

	nationName := "No Nation"
	if resident.Nation.Name != nil {
		nationName = *resident.Nation.Name
	}

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

	embed := &dgo.MessageEmbed{
		Type:  dgo.EmbedTypeRich,
		Color: 7419530,
		Thumbnail: &dgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", resident.UUID),
		},
		Title: fmt.Sprintf("Player Information | `%s`", resName),
		Fields: []*dgo.MessageEmbedField{
			affiliationField,
			EmbedField("Balance", utils.HumanizedSprintf("`%.0f`G", resident.Stats.Balance), false),
			EmbedField("Status", status, true),
			EmbedField("Last Online", fmt.Sprintf("<t:%d:R>", lastOnlineTs), true),
			EmbedField("Registered", fmt.Sprintf("<t:%d:F>", registeredTs), true),
		},
	}

	// Alias differs from name (has surname and/or title)
	if alias != resName {
		field := EmbedField("Alias", alias, false)
		embed.Fields = append([]*dgo.MessageEmbedField{field}, embed.Fields...) // Prepend
	}

	return embed
}

func CreateTownEmbed(town objs.TownInfo) *dgo.MessageEmbed {
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

func CreateNationEmbed(nation objs.NationInfo) *dgo.MessageEmbed {
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

func CreateStaffEmbed() (*dgo.MessageEmbed, error) {
	var onlineStaff []string
	var errors []error

	ids := GetStaffIds()
	players, err := oapi.QueryPlayers(ids...)

	// Calls specified func for every slice element in parallel.
	lop.ForEach(players, func(p objs.PlayerInfo, _ int) {
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
