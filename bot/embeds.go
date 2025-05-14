package bot

import (
	"emcsrw/oapi"
	"emcsrw/utils"
	"fmt"
	"strconv"
	"strings"

	lo "github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	log "github.com/sirupsen/logrus"

	dgo "github.com/bwmarrin/discordgo"
)

func SendComplex(discord *dgo.Session, message *dgo.MessageCreate, embed *dgo.MessageSend) {
	_, err := discord.ChannelMessageSendComplex(message.ChannelID, embed)
	if err != nil {
		log.Error(err)
	}
}

func CheckAlphanumeric(str string) string {
	return lo.Ternary(utils.ContainsNonAlphanumeric(str), "", str)
}

func CreateResidentEmbed(discord *dgo.Session, message *dgo.MessageCreate, args []string) (*dgo.MessageEmbed, error) {
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

		resTitle := CheckAlphanumeric(resident.Title)
		resSurname := CheckAlphanumeric(resident.Surname)
		resName := resident.Name

		if resTitle != "" {
			resName = resTitle + " " + resName
		}

		if resSurname != "" {
			resName += (" " + resSurname)
		}

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
				Name:    message.Author.Username,
				IconURL: message.Author.AvatarURL(""),
			},
			Thumbnail: &dgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", resident.UUID),
			},
		}, nil
	}

	return nil, err
}

func CreateTownEmbed(discord *dgo.Session, message *dgo.MessageCreate, args []string) (*dgo.MessageEmbed, error) {
	town, err := oapi.Town(args[2])
	if err == nil {
		residents := strings.Join(town.Residents, ", ")

		foundedTs := utils.FormatTimestamp(town.Timestamps.Registered)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		townTitle := fmt.Sprintf("Town | %s", *town.Name)
		if town.Nation != "" {
			townTitle += fmt.Sprintf(" (%s)", town.Nation)
		}

		return &dgo.MessageEmbed{
			Type:  dgo.EmbedTypeRich,
			Title: townTitle,
			Fields: []*dgo.MessageEmbedField{
				EmbedField("Founder", town.Founder, true),
				EmbedField("Date Founded", dateFounded, true),
				EmbedField("Mayor", town.Mayor, false),
				EmbedField("Area", fmt.Sprintf("%d / %d Chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
				EmbedField("Balance", fmt.Sprintf("%.0fG", town.Stats.Balance), true),
				EmbedField("Residents", fmt.Sprintf("```%s```", residents), false),
			},
			Color: utils.HexToInt(town.HexColor),
			Author: &dgo.MessageEmbedAuthor{
				Name:    message.Author.Username,
				IconURL: message.Author.AvatarURL(""),
			},
		}, nil
	}

	return nil, err
}

func CreateNationEmbed(discord *dgo.Session, message *dgo.MessageCreate, args []string) (*dgo.MessageEmbed, error) {
	nation, err := oapi.Nation(args[2])
	if err == nil {
		foundedTs := strconv.FormatFloat(nation.Timestamps.Registered/1000, 'f', 0, 64)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		return &dgo.MessageEmbed{
			Type:  dgo.EmbedTypeRich,
			Title: fmt.Sprintf("Nation | %s", *nation.Name),
			Fields: []*dgo.MessageEmbedField{
				EmbedField("King", nation.King, true),
				EmbedField("Capital", nation.Capital, true),
				EmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", nation.Spawn.X, nation.Spawn.Z, nation.Spawn.X, nation.Spawn.Y, nation.Spawn.Z), true),
				EmbedField("Date Founded", dateFounded, true),
				EmbedField("Area", fmt.Sprintf("%d Chunks", nation.Stats.NumTownBlocks), true),
				EmbedField("Balance", fmt.Sprintf("%.0fG", nation.Stats.Balance), true),
				EmbedField("Residents", fmt.Sprintf("```%d```", len(nation.Residents)), false),
			},
			Color: utils.HexToInt(nation.HexColor),
			Author: &dgo.MessageEmbedAuthor{
				Name:    message.Author.Username,
				IconURL: message.Author.AvatarURL(""),
			},
		}, nil
	}

	return nil, err
}

func CreateStaffEmbed(discord *dgo.Session, message *dgo.MessageCreate, args []string) (*dgo.MessageEmbed, error) {
	var onlineStaff []string
	var errors []error

	// Iterates over the collection, calling func in parallel.
	lop.ForEach(StaffIds, func(uuid string, _ int) {
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

	return &dgo.MessageEmbed{
		Type:        dgo.EmbedTypeRich,
		Title:       "Staff List | Online",
		Description: fmt.Sprintf("```%s```", content),
		Color:       15844367,
		Author: &dgo.MessageEmbedAuthor{
			Name:    message.Author.Username,
			IconURL: message.Author.AvatarURL(""),
		},
	}, nil
}
