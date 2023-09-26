package bot

import (
	"emcs-rewritten/api/nations"
	"emcs-rewritten/api/residents"
	"emcs-rewritten/api/towns"
	"emcs-rewritten/structs"
	"emcs-rewritten/utils"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var BotToken string

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

func CreateResidentEmbed(discord *discordgo.Session, message *discordgo.MessageCreate, args []string) (*discordgo.MessageSend, error) {
	resident, err := residents.Get(args[2])

	if err == nil {
		registeredTs := utils.FormatTimestamp(resident.Timestamps.Registered)
		lastOnlineTs := utils.FormatTimestamp(resident.Timestamps.LastOnline)

		status := "Offline"
		if resident.Status.Online { status = "Online" }
		
		town := resident.Town
		if town == "" { town = "No Town" }

		nation := resident.Nation
		if nation == "" { nation = "No Nation" }
		
		re := regexp.MustCompile("<.*?>")
		resName := resident.Name

		if resTitle := re.ReplaceAllString(resident.Title, "")
			resTitle != "" { resName = resTitle + " " + resName }

		if resSurname := re.ReplaceAllString(resident.Surname, "")
			resSurname != "" { resName = resName + " " + resSurname }

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Resident | `%s`", resName),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("Affiliation", fmt.Sprintf("%s (%s)", town, nation), false),
					EmbedField("Balance", fmt.Sprintf("%.0fG", resident.Stats.Balance), false),
					EmbedField("Status", status, true),
					EmbedField("Last Online", fmt.Sprintf("<t:%s:R>", lastOnlineTs), true),
					EmbedField("Registered", fmt.Sprintf("<t:%s:F>", registeredTs), true),
				},
				Color: 7419530,
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

func CreateTownEmbed(discord *discordgo.Session, message *discordgo.MessageCreate, args []string) (*discordgo.MessageSend, error) {
	town, err := towns.Get(args[2])
	
	if err == nil {
		residents := strings.Join(town.Residents, ", ")

		foundedTs := utils.FormatTimestamp(town.Timestamps.Registered)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		townTitle := fmt.Sprintf("Town | %s", *town.Name)
		if town.Nation != "" {
			townTitle += fmt.Sprintf(" (%s)", town.Nation)
		}

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: townTitle,
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("Founder", town.Founder, true),
					EmbedField("Date Founded", dateFounded, true),
					EmbedField("Mayor", town.Mayor, false),
					EmbedField("Area", fmt.Sprintf("%d / %d Chunks", town.Stats.NumTownBlocks, town.Stats.MaxTownBlocks), true),
					EmbedField("Balance", fmt.Sprintf("%.0fG", town.Stats.Balance), true),
					EmbedField("Residents", fmt.Sprintf("```%s```", residents), false),
				},
				Color: utils.HexToInt(town.HexColor),
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

func CreateNationEmbed(discord *discordgo.Session, message *discordgo.MessageCreate, args []string) (*discordgo.MessageSend, error) {
	nation, err := nations.Get(args[2])

	if err == nil {
		foundedTs := strconv.FormatFloat(nation.Timestamps.Registered / 1000, 'f', 0, 64)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Nation | %s", *nation.Name),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("King", nation.King, true),
					EmbedField("Capital", nation.Capital, true),
					EmbedField("Location", fmt.Sprintf("[%.0f, %.0f](https://earthmc.net/map/aurora/?worldname=earth&mapname=flat&zoom=5&x=%f&y=%f&z=%f)", nation.Spawn.X, nation.Spawn.Z, nation.Spawn.X, nation.Spawn.Y, nation.Spawn.Z), true),
					EmbedField("Date Founded", dateFounded, true),
					EmbedField("Area", fmt.Sprintf("%d Chunks", nation.Stats.NumTownBlocks), true),
					EmbedField("Balance", fmt.Sprintf("%.0fG", nation.Stats.Balance), true),
					EmbedField("Residents", fmt.Sprintf("```%d```", len(nation.Residents)), false),
				},
				Color: utils.HexToInt(nation.HexColor),
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

func CreateStaffEmbed(discord *discordgo.Session, message *discordgo.MessageCreate, args []string) (*discordgo.MessageSend, error) {
	var (
        onlineStaff []string
		res 		structs.ResidentInfo
		err         error
    )

    wg := sync.WaitGroup{}

	for _, uuid := range StaffIds {
		wg.Add(1)

		go func(uuid string) {
			res, err = residents.Get(uuid)
	
			if err != nil {
				fmt.Println(err)
				return
			}
	
			if res.Status.Online {
				onlineStaff = append(onlineStaff, res.Name)
			}
	
			defer wg.Done()
		} (uuid)
	}

	wg.Wait()

	content := "None"
	if len(onlineStaff) > 0 { 
		content = strings.Join(onlineStaff, ", ") 
	}

	staffEmbed := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Staff List | Online",
			Description: fmt.Sprintf("```%s```", content),
			Color:       15844367,
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
	if message.Author.ID == discord.State.User.ID { 
		return 
	}

	msgContent := message.Content
	if strings.HasPrefix(msgContent, "emcs") == true { 
		return 
	}

	args := strings.Split(message.Content, " ")
	cmd := strings.ToLower(args[1])

	switch {
		case cmd == "stafflist", cmd == "staff": {
			embed := &discordgo.MessageSend{
				Embeds: []*discordgo.MessageEmbed{{
					Type:        discordgo.EmbedTypeRich,
					Title:       "Staff List",
					Description: fmt.Sprintf("```%s```", strings.Join(StaffNames, ", ")),
					Color:       15844367,
					Author: &discordgo.MessageEmbedAuthor{
						Name:    message.Author.Username,
						IconURL: message.Author.AvatarURL(""),
					},
				}},
			}

			sendEmbed(discord, message, embed)
		}

		case cmd == "onlinestaff", cmd == "ostaff": {
			embed, err := CreateStaffEmbed(discord, message, args)

			if (err != nil) {
				errMsg := "Could not fetch staff list!\nAn error occurred during the request."
				discord.ChannelMessageSend(message.ChannelID, errMsg)
				
				return
			}
			
			sendEmbed(discord, message, embed)
		}

		case cmd == "resident", cmd == "res", cmd == "r": {
			embed, err := CreateResidentEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch resident!\nThe resident does not exist or an error occurred."
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			sendEmbed(discord, message, embed)
		}

		case cmd == "town", cmd == "t": {
			embed, err := CreateTownEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch town!\nThe town does not exist or an error occurred."
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			sendEmbed(discord, message, embed)
		}

		case cmd == "nation", cmd == "n": {
			embed, err := CreateNationEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch nation!\nThe nation does not exist or an error occurred!"
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			sendEmbed(discord, message, embed)
		}
	}
}