package bot

import (
	"emcs-rewritten/classes/towns"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
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

func CreateTownEmbed(
	discord *discordgo.Session, 
	message *discordgo.MessageCreate, 
	args []string,
) (*discordgo.MessageSend, error) {
	town, err := towns.Get(args[2])
	if err == nil {
		area := fmt.Sprintf(
			"Chunks claimed: `%s` / `%s`", 
			fmt.Sprint(town.Stats.NumTownBlocks),
			fmt.Sprint(town.Stats.MaxTownBlocks),
		)

		bal := fmt.Sprint(town.Stats.Balance)
		residents := strings.Join(town.Residents, ", ")

		foundedTs := strconv.FormatFloat(town.Timestamps.Registered, 'f', -1, 64)
		dateFounded := fmt.Sprintf("<t:%s:R>", foundedTs)

		embed := &discordgo.MessageSend{
			Embeds: [] *discordgo.MessageEmbed{{
				Type: discordgo.EmbedTypeRich,
				Title: fmt.Sprintf("Town | %s", town.Name),
				Fields: []*discordgo.MessageEmbedField{
					EmbedField("Mayor", town.Mayor, true),
					EmbedField("Founder", town.Founder, true),
					EmbedField("Date Founded", dateFounded, true),
					EmbedField("Area", area, true),
					EmbedField("Balance", bal, true),
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

func messageCreate(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == discord.State.User.ID { return }

	args := strings.Split(message.Content, " ")
	if (strings.ToLower(args[0]) != "emcs") { return }

	cmd := strings.ToLower(args[1])

	switch {
		case cmd == "staff": {
			staffList := []string {
				"Fix", "1212rah", "Proser", "Barbay1", "Fruitloopins", "KarlOfDuty", "Oce_78", "earlgreyteabag", "WTDpuddles", 
				"PolkadotBlueBear", "32Andrew", "Gris_", "CorruptedGreed", "Shia_chan", "Unbated", "KeijoDPutt", "404Fiji", 
				"Professor__Pro", "Coblobster", "Superaktif", "__Zlatan__", "RlZ58", "BigshotWarrior", "Mednis","Creightor", 
				"linkeron1", "XxSlayerMCxX", "dopecello", "ShindoiASU", "Lion_Of_Judah12", "aas5aa_OvO", "Warriorrr", "Masrain", 
				"Lucas2011", "RAWRyoutube", "wishful_", "Aehmett", "fabinostalgia", "yelune", "SuperHappyBros", "32Breach",
			}

			listStr := fmt.Sprintf("```%s```", strings.Join(staffList, ", "))
			discord.ChannelMessageSend(message.ChannelID, listStr)
		}
		case cmd == "town": {
			townEmbed, err := CreateTownEmbed(discord, message, args)
			
			if (err != nil) {
				discord.ChannelMessageSend(
					message.ChannelID, 
					fmt.Sprintf("Could not fetch town!\nAn error occurred during the request!"),
				)

				return
			}
			
			discord.ChannelMessageSendComplex(message.ChannelID, townEmbed)
		}
	}
}