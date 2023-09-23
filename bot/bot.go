package bot

import (
	"emcs-rewritten/api/residents"
	"emcs-rewritten/api/towns"
	"emcs-rewritten/api/nations"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var BotToken string

// Last updated - 23/09/2023
var staffList = []string {
	"b4d2215c-e47b-4f39-a3e3-e6726e0bc596", "f17d77ab-aed4-44e7-96ef-ec9cd473eda3",
	"ea798ca9-1192-4334-8ef5-f098bdc9cb39", "fed0ec4a-f1ad-4b97-9443-876391668b34",
	"e25ad129-fe8a-4306-b1af-1dee1ff59841", "8f90a970-90de-407d-b4e7-0d9fde10f51a",
	"7b589f55-c5e2-41b7-89fc-13eb8781058e", "96d55845-c99f-447d-923d-1e652ab50bb1",
	"7f551e7c-082c-4533-b484-7435d9941d0d", "bae75988-00f4-41ce-b902-abc21fcb8978",
	"d32c9786-17e6-43a2-8f83-8f95a6655893", "c14bb5d7-6d51-4e3f-a388-36d6a5992272",
	"9d3857a7-9fb7-4232-ae5e-c66d6ca1f621", "ad978482-b062-472b-a168-4d317af0f72a",
	"94a4c5ea-027f-4509-a434-66320c29948a", "16b5a2e4-04a3-40f7-8311-4bffeccd23ba",
	"0b992c62-587d-45ea-8821-4f3b39bb6cc4", "0bacd488-bc41-4f76-ba8b-50dc843efe49",
	"2bf8f573-f2e8-4d2c-b1e9-072bf052722a", "2de5dbaa-27aa-4c1a-82f6-65f4ed966592",
	"376d4c35-7ffb-480b-9bd3-bc1ef1bd09f7", "379f1b2b-0b53-4ea4-b0b2-14b6e4777983",
	"ea0c4d98-106d-416c-9647-06203f861e1b", "25e7015d-8ad0-4c27-bb4e-a31b4c17e979",
	"479d37aa-229f-41a4-a49f-784c29e1fe65", "a87996b7-243a-40ec-af03-f7604a5ae97b",
	"25c9762f-df85-45d2-8edd-be83c09eec34", "7a9da17c-834d-4f45-b4ce-07ea04274a12",
	"9449b781-a702-484a-934d-9ed351bcebd4", "0c16f3f8-59c2-413e-9327-8f4b1f7f44cf",
	"7f235a6f-2988-4967-ba5d-adbe2781728f", "a82437de-882d-43c2-9e12-3355e8783c9d",
	"3947e217-95ae-4952-93b7-6b4004b54f40", "ad0a2de0-1e73-401d-967c-f5b155dc6ad1",
	"0ef76084-7c86-4115-b1a5-a9c4e9bc89a6", "c99d4771-f844-4eff-b2ac-aee086d15bbe",
	"19c5aede-d9af-4234-9a21-b956ae99b0ba", "c200e96f-834e-40b7-81d5-e27fd3107ecd",
	"2cbaad0b-ed5b-453e-a6b0-ec4b88ca43cd", "4452780b-034d-4d46-b427-717ce3f5acda",
}


func Run() {
	// Create new Discord Session
	discord, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(messageCreate)

	// Open session
	discord.Open()
	defer discord.Close()

	// Run until code is terminated
	fmt.Println("Bot running...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}


func EmbedField(name string, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField {
		Name: name,
		Value: value,
		Inline: inline,
	}
}


func CreateResidentEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	args []string,
) (*discordgo.MessageSend, error) {
	resident, err := residents.Get(args[2])
	if err == nil {

		registeredTs := strconv.FormatFloat(resident.Timestamps.Registered / 1000, 'f', 0, 64)
		dateRegistered := fmt.Sprintf("<t:%s:F>", registeredTs)

		lastOnlineTs := strconv.FormatFloat(resident.Timestamps.LastOnline / 1000, 'f', 0, 64)
		dateLastOnline := fmt.Sprintf("<t:%s:R>", lastOnlineTs)

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

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Resident | `%s %s %s`", resident.Title, resident.Name, resident.Surname),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("Affiliation", fmt.Sprintf("%s (%s)", town, nation), true),
					EmbedField("Balance", fmt.Sprintf("%.0fG", resident.Stats.Balance), true),
					EmbedField("Status", status, true),
					EmbedField("Registered", dateRegistered, true),
					EmbedField("Last Online", dateLastOnline, true),
				},
				Color: 5763719,
				Author: &discordgo.MessageEmbedAuthor{
					Name: message.Author.Username,
					IconURL: message.Author.AvatarURL(""),
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: fmt.Sprintf("https://visage.surgeplay.com/bust/%s.png?width=230&height=230", resident.UUID),
				},
				
			}},
		}

		return embed, nil
	}

	return nil, err
}


func CreateTownEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	args []string,
) (*discordgo.MessageSend, error) {
	town, err := towns.Get(args[2])
	if err == nil {

		residents := strings.Join(town.Residents, ", ")

		nation := town.Nation
		if nation == "" {
			nation = "No Nation"
		}

		foundedTs := strconv.FormatFloat(town.Timestamps.Registered / 1000, 'f', 0, 64)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Town | %s (%s)", town.Name, nation),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("Mayor", town.Mayor, true),
					EmbedField("Founder", town.Founder, true),
					EmbedField("Date Founded", dateFounded, true),
					EmbedField("Area", fmt.Sprintf("`%d` / `%d` chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
					EmbedField("Balance", fmt.Sprintf("%.0fG", town.Stats.Balance), true),
					EmbedField("Residents", fmt.Sprintf("```%s```", residents), false),
					//EmbedField("", "", false),
				},
				Color: 5763719,
				Author: &discordgo.MessageEmbedAuthor{
					Name: message.Author.Username,
					IconURL: message.Author.AvatarURL(""),
				},
			}},
		}

		return embed, nil
	}

	return nil, err
}


func CreateNationEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	args []string,
) (*discordgo.MessageSend, error) {
	nation, err := nations.Get(args[2])
	if err == nil {

		foundedTs := strconv.FormatFloat(nation.Timestamps.Registered / 1000, 'f', 0, 64)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Nation | %s", nation.Name),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("King", nation.King, true),
					EmbedField("Capital", nation.Capital, true),
					EmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", nation.Spawn.X, nation.Spawn.Z, nation.Spawn.X, nation.Spawn.Y, nation.Spawn.Z), true),
					EmbedField("Date Founded", dateFounded, true),
					EmbedField("Area", fmt.Sprintf("%d chunks", nation.Stats.NumTownBlocks), true),
					EmbedField("Balance", fmt.Sprintf("%.0fG", nation.Stats.Balance), true),
					EmbedField("Residents", fmt.Sprintf("```%d```", len(nation.Residents)), false),
				},
				Color: 5763719,
				Author: &discordgo.MessageEmbedAuthor{
					Name: message.Author.Username,
					IconURL: message.Author.AvatarURL(""),
				},
			}},
		}

		return embed, nil
	}

	return nil, err
}


func CreateStaffEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	args []string,
) (*discordgo.MessageSend, error) {
	onlineStaff := []string {}

	for _, elem := range staffList {
		res, err := residents.Get(elem)

		if err != nil {
			discord.ChannelMessageSend(
				message.ChannelID,
				"Could not fetch staff list!\nAn error occurred during the request!",
			)
			
			return nil, err
		}

		if res.Status.Online {
			onlineStaff = append(onlineStaff, res.Name)
		}
	}

	staffEmbed := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Online staff",
			Description: fmt.Sprintf("```%s```", strings.Join(onlineStaff, ", ")),
			Color:       5763719,
			Author: &discordgo.MessageEmbedAuthor{
				Name:    message.Author.Username,
				IconURL: message.Author.AvatarURL(""),
			},
		}},
	}

	return staffEmbed, nil
}

func sendEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	embed *discordgo.MessageSend,
) {
	_, err := discord.ChannelMessageSendComplex(message.ChannelID, embed)

	if err != nil {
		log.Error(err)
	}
}




func messageCreate(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == discord.State.User.ID { return }

	args := strings.Split(message.Content, " ")
	if (strings.ToLower(args[0]) != "emcs") { return }

	cmd := strings.ToLower(args[1])

	switch {


		case cmd == "staff": {
			embed, err := CreateStaffEmbed(discord, message, args)

			if (err != nil) {
				discord.ChannelMessageSend(
					message.ChannelID,
					"Could not fetch staff list!\nAn error occurred during the request!",
				)
				
				return
			}
			
			sendEmbed(discord, message, embed)
		}


		case cmd == "resident", cmd == "res", cmd == "r": {
			embed, err := CreateResidentEmbed(discord, message, args)
			
			if (err != nil) {
				discord.ChannelMessageSend(
					message.ChannelID,
					"Could not fetch resident!\nAn error occurred during the request!",
				)

				return
			}
			
			sendEmbed(discord, message, embed)
		}


		case cmd == "town", cmd == "t": {
			embed, err := CreateTownEmbed(discord, message, args)
			
			if (err != nil) {
				discord.ChannelMessageSend(
					message.ChannelID,
					"Could not fetch town!\nAn error occurred during the request!",
				)

				return
			}
			
			sendEmbed(discord, message, embed)
		}


		case cmd == "nation", cmd == "n": {
			embed, err := CreateNationEmbed(discord, message, args)
			
			if (err != nil) {
				discord.ChannelMessageSend(
					message.ChannelID,
					"Could not fetch nation!\nAn error occurred during the request!",
				)

				return
			}
			
			sendEmbed(discord, message, embed)
		}

	}
}