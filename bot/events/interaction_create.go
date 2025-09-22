package events

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/slashcommands"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"fmt"
	"log"
	"math/rand"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateApplicationCommand recovered from a panic.\n%v\n%s", err, debug.Stack())
			ReplyWithError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.UserFromInteraction(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmdType := i.ApplicationCommandData().CommandType
	cmd := slashcommands.All()[cmdName]

	start := time.Now()
	err := cmd.Execute(s, i)
	elapsed := time.Since(start)

	success := false
	if err != nil {
		log.Printf("'%s' failed to execute command /%s:\n%v\n\n", author.Username, cmdName, err)
	} else {
		log.Printf("'%s' successfully executed command /%s (took: %s)\n", author.Username, cmdName, elapsed)
		success = true
	}

	// TODO: Maybe combine get/put into single update transaction. Rn this is View&Get + Update&Set
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	// usage, err := database.GetUserUsage(db, author.ID)
	// if err != nil {
	// 	fmt.Printf("\ndb error occurred. could not get usage for user: %s (%s)\n%v", author.Username, author.ID, err)
	// 	return
	// }

	// Add a new command usage entry to history.
	// usage.CommandHistory[cmdName] = append(usage.CommandHistory[cmdName], database.CommandEntry{
	// 	Type:      uint8(cmdType),
	// 	Timestamp: time.Now().Unix(),
	// 	Success:   success,
	// })

	if cmdName != "usage" {
		err = database.UpdateUserUsage(db, author.ID, cmdName, database.CommandEntry{
			Type:      uint8(cmdType),
			Timestamp: time.Now().Unix(),
			Success:   success,
		})
	}

	if err != nil {
		log.Printf("\ndb error occurred. could not update usage for user: %s (%s)\n%v", author.Username, author.ID, err)
	}
}

// List of message components: https://discord.com/developers/docs/components/reference#component-object-component-types
func OnInteractionCreateMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
}

func OnInteractionCreateModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnInteractionCreateModalSubmit recovered from a panic.\n%v\n%s", err, debug.Stack())
			ReplyWithError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionModalSubmit {
		return
	}

	data := i.ModalSubmitData()
	if data.CustomID == "alliance_creator_modal" {
		HandleAllianceCreatorModal(s, i.Interaction, data)
	}
}

func HandleAllianceCreatorModal(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ModalSubmitInteractionData) {
	inputs := ModalInputsToMap(i)

	label := inputs["label"]
	ident := inputs["identifier"]
	nations := inputs["nations"]

	createdAlliance := &database.Alliance{
		UUID:       generateAllianceUUID(),
		Identifier: ident,
		Label:      label,
		OwnNations: strings.Split(nations, ", "),
	}

	fmt.Print(utils.Prettify(createdAlliance))
	//fmt.Printf("Label: %s, Ident: %s\nNations: %s\n", label, ident, nationsInput)
}

func ModalInputsToMap(i *discordgo.Interaction) map[string]string {
	inputs := make(map[string]string)
	for _, row := range i.ModalSubmitData().Components {
		if actionRow, ok := row.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if input, ok := comp.(*discordgo.TextInput); ok {
					inputs[input.CustomID] = input.Value
				}
			}
		}
	}
	return inputs
}

func generateAllianceUUID() uint64 {
	created := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))      // Safe to cast to uint since Intn returns 0-n anyway.
	return (created << 16) | suffix
}

func ReplyWithError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot attempted to panic during this command. Please report the following error.\n" + errStr

	// Not already deferred, reply.
	err = discordutil.SendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		discordutil.FollowUpContentEphemeral(s, i, content)
	}
}
