package bot

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

func Run(botToken string) {
	// Create new Discord Session
	discord, err := discordgo.New("Bot " + botToken)
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

const RED, YELLOW int = 8858420, 15844367
func messageCreate(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == discord.State.User.ID { 
		return 
	}

	msgContent := message.Content
	if strings.HasPrefix(msgContent, "emcs") == false { 
		return 
	}

	args := strings.Split(message.Content, " ")
	cmd := strings.ToLower(args[1])

	switch {
		case cmd == "stafflist", cmd == "staff": {
			author := &discordgo.MessageEmbedAuthor{
				Name:    	message.Author.Username,
				IconURL: 	message.Author.AvatarURL(""),
			}

			embed := &discordgo.MessageSend{
				Embeds: []*discordgo.MessageEmbed{{
					Type: discordgo.EmbedTypeRich,
					Title: "Staff List",
					Description: fmt.Sprintf("```%s```", strings.Join(StaffNames, ", ")),
					Color: YELLOW,
					Author:	author,
				}},
			}

			SendComplex(discord, message, embed)
		}

		case cmd == "onlinestaff", cmd == "ostaff": {
			embed, err := CreateStaffEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "The following error occurred during your request:"
				desc := fmt.Sprintf("%s\n\n```%s```", errMsg, err.Error())

				embed = &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{{
						Type: discordgo.EmbedTypeRich,
						Title: "Could not fetch online staff!",
						Description: desc,
						Color: RED,
					}},
				}
			}
			
			SendComplex(discord, message, embed)
		}

		case cmd == "resident", cmd == "res", cmd == "r": {
			embed, err := CreateResidentEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch resident!\nThe resident does not exist or an error occurred."
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			SendComplex(discord, message, embed)
		}

		case cmd == "town", cmd == "t": {
			embed, err := CreateTownEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch town!\nThe town does not exist or an error occurred."
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			SendComplex(discord, message, embed)
		}

		case cmd == "nation", cmd == "n": {
			embed, err := CreateNationEmbed(discord, message, args)
			
			if (err != nil) {
				errMsg := "Could not fetch nation!\nThe nation does not exist or an error occurred!"
				discord.ChannelMessageSend(message.ChannelID, errMsg)

				return
			}
			
			SendComplex(discord, message, embed)
		}
	}
}