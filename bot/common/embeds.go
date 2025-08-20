package common

import (
	"emcsrw/oapi"
	"emcsrw/oapi/objs"
	"emcsrw/utils"
	"fmt"
	"strconv"
	"strings"

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

func CreateResidentEmbed(i *dgo.Interaction, args []string) (*dgo.MessageEmbed, error) {
	resident, err := oapi.Resident(args[2])
	if err == nil {
		registeredTs := utils.FormatTimestamp(resident.Timestamps.Registered)
		lastOnlineTs := utils.FormatTimestamp(*resident.Timestamps.LastOnline)

		status := "Offline"
		if resident.Status.Online {
			status = "Online"
		}

		town := resident.Town
		if town == "" {
			town = "No Town"
		}

		nation := resident.Nation
		if nation == "" {
			nation = "No Nation"
		}

		resTitle := utils.CheckAlphanumeric(resident.Title)
		resSurname := utils.CheckAlphanumeric(resident.Surname)
		resName := resident.Name

		if resTitle != "" {
			resName = resTitle + " " + resName
		}

		if resSurname != "" {
			resName += (" " + resSurname)
		}

		author := utils.UserFromInteraction(i)
		return &dgo.MessageEmbed{
			Type:  dgo.EmbedTypeRich,
			Title: fmt.Sprintf("Resident | `%s`", resName),
			Fields: []*dgo.MessageEmbedField{
				EmbedField("Affiliation", fmt.Sprintf("%s (%s)", town, nation), false),
				EmbedField("Balance", fmt.Sprintf("%.0fG", resident.Stats.Balance), false),
				EmbedField("Status", status, true),
				EmbedField("Last Online", fmt.Sprintf("<t:%s:R>", lastOnlineTs), true),
				EmbedField("Registered", fmt.Sprintf("<t:%s:F>", registeredTs), true),
			},
			Color: 7419530,
			Author: &dgo.MessageEmbedAuthor{
				Name:    author.Username,
				IconURL: author.AvatarURL(""),
			},
			Thumbnail: &dgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", resident.UUID),
			},
		}, nil
	}

	return nil, err
}

func CreateTownEmbed(i *dgo.Interaction, town objs.TownInfo) *dgo.MessageEmbed {
	townTitle := fmt.Sprintf("Town Information | %s", town.Name)
	if town.Nation.Name != "" {
		townTitle += fmt.Sprintf(" (%s)", town.Nation.Name)
	}

	foundedTs := utils.FormatTimestamp(town.Timestamps.Registered)
	foundedDate := fmt.Sprintf("<t:%s:R>", foundedTs)

	overclaimShield := SHIELD_EMOJIS.RED + " Inactive"
	if town.Status.HasOverclaimShield {
		overclaimShield = SHIELD_EMOJIS.GREEN + " Active"
	}

	desc := ""
	if town.Board != DEFAULT_TOWN_BOARD {
		desc = fmt.Sprintf("*%s*", town.Board)
	}

	author := utils.UserFromInteraction(i)
	return &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       townTitle,
		Description: desc,
		Fields: []*dgo.MessageEmbedField{
			EmbedField("Founder", town.Founder, true),
			EmbedField("Date Founded", foundedDate, true),
			EmbedField("Mayor", town.Mayor.Name, false),
			EmbedField("Area", fmt.Sprintf("%d / %d Chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
			EmbedField("Balance", fmt.Sprintf("%.0fG", town.Bal()), true),
			EmbedField("Residents", fmt.Sprintf("`%d`", town.Stats.NumResidents), false),
			EmbedField("Overclaimed", town.OverclaimedString(), false),
			EmbedField("Overclaimed Shield", overclaimShield, false),
		},
		Color: utils.HexToInt("2ecc71"), // GREEN
		Author: &dgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
	}
}

func CreateNationEmbed(i *dgo.Interaction, nation objs.NationInfo) *dgo.MessageEmbed {
	foundedTs := strconv.FormatFloat(nation.Timestamps.Registered/1000, 'f', 0, 64)
	dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

	spawn := nation.Spawn

	author := utils.UserFromInteraction(i)
	return &dgo.MessageEmbed{
		Type:  dgo.EmbedTypeRich,
		Title: fmt.Sprintf("Nation | %s", nation.Name),
		Fields: []*dgo.MessageEmbedField{
			EmbedField("King", nation.King.Name, true),
			EmbedField("Capital", nation.Capital.Name, true),
			EmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", spawn.X, spawn.Z, spawn.X, spawn.Y, spawn.Z), true),
			EmbedField("Date Founded", dateFounded, true),
			EmbedField("Area", fmt.Sprintf("%d Chunks", nation.Stats.NumTownBlocks), true),
			EmbedField("Balance", fmt.Sprintf("%.0fG", nation.Stats.Balance), true),
			EmbedField("Residents", fmt.Sprintf("`%d`", nation.Stats.NumResidents), false),
		},
		Color: nation.FillColourInt(),
		Author: &dgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
	}
}

func CreateStaffEmbed(i *dgo.Interaction, args []string) (*dgo.MessageEmbed, error) {
	var onlineStaff []string
	var errors []error

	// Iterates over the collection, calling func in parallel.
	lop.ForEach(GetStaffIds(), func(uuid string, _ int) {
		res, err := oapi.Resident(uuid)

		if err != nil {
			fmt.Println(err)
			errors = append(errors, err)

			return
		}

		if res.Status.Online {
			onlineStaff = append(onlineStaff, res.Name)
		}
	})

	if len(errors) > 0 {
		return nil, errors[0]
	}

	content := "None"
	if len(onlineStaff) > 0 {
		content = strings.Join(onlineStaff, ", ")
	}

	author := utils.UserFromInteraction(i)
	return &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       "Staff List | Online",
		Description: fmt.Sprintf("```%s```", content),
		Color:       15844367,
		Author: &dgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
	}, nil
}
