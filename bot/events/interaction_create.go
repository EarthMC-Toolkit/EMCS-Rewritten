package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/bot/store"
	"emcsrw/shared"
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
			ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.GetInteractionAuthor(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmdType := i.ApplicationCommandData().CommandType
	cmd := slashcommands.All()[cmdName]

	start := time.Now()
	err := cmd.Execute(s, i)
	elapsed := time.Since(start)

	success := err == nil
	fmt.Println()
	if success {
		log.Printf("'%s' successfully executed command /%s (took: %s)\n", author.Username, cmdName, elapsed)
	} else {
		log.Printf("'%s' failed to execute command /%s:\n%v\n\n", author.Username, cmdName, err)
		// TODO: Maybe send error message here like we do with panic
	}

	if cmdName != "usage" {
		mdb, err := store.GetMapDB(shared.ACTIVE_MAP)
		if err != nil {
			fmt.Println()
			log.Printf("error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}

		e := store.UsageCommandEntry{
			Type:      uint8(cmdType),
			Timestamp: time.Now().Unix(),
			Success:   success,
		}

		if err := store.UpdateUsageForUser(mdb, author, cmdName, e); err != nil {
			fmt.Println()
			log.Printf("error updating usage for user: %s (%s)\n%v", author.Username, author.ID, err)
			return
		}
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
			ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionModalSubmit {
		return
	}

	data := i.ModalSubmitData()
	if data.CustomID == "alliance_creator_modal" {
		err := handleAllianceCreatorModal(s, i.Interaction, data)
		if err != nil {
			ReplyWithError(s, i.Interaction, err)
			return
		}
	}
}

// Handles the submission of the modal for creating an alliance.
func handleAllianceCreatorModal(s *discordgo.Session, i *discordgo.Interaction, _ discordgo.ModalSubmitInteractionData) error {
	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	inputs := ModalInputsToMap(i)

	label := inputs["label"]
	ident := inputs["identifier"]
	nations := inputs["nations"]

	representative := discordutil.GetInteractionAuthor(i)

	// Uncomment if using representative input
	// if representative == nil {
	// 	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
	// 		Content: "Could not determine representative user.",
	// 		Flags:   discordgo.MessageFlagsEphemeral,
	// 	})

	// 	return nil
	// }

	alliance := store.Alliance{
		UUID:             generateAllianceID(),
		Identifier:       ident,
		Label:            label,
		RepresentativeID: &representative.ID,
		OwnNations:       strings.Split(nations, ", "),
	}

	//fmt.Print(utils.Prettify(alliance))

	allianceStore.SetKey(strings.ToLower(ident), alliance)

	discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Content: "Successfully created alliance:",
		Embeds: []*discordgo.MessageEmbed{
			shared.NewAllianceEmbed(s, &alliance),
		},
	})

	return nil
}

func ModalInputsToMap(i *discordgo.Interaction) map[string]string {
	inputs := make(map[string]string)
	for _, row := range i.ModalSubmitData().Components {
		actionRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue // Must not be an action row, we don't care.
		}

		// Gather values of all text input components in this row.
		for _, comp := range actionRow.Components {
			if input, ok := comp.(*discordgo.TextInput); ok {
				inputs[input.CustomID] = input.Value
			}
		}
	}

	return inputs
}

func generateAllianceID() uint64 {
	createdTs := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))        // Safe to cast to uint since Intn returns 0-n anyway.
	return (createdTs << 16) | suffix
}

func ReplyWithPanicError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot attempted to fatally crash during this command! Please report the following error.\n" + errStr

	// Not already deferred, reply.
	_, err = discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		discordutil.FollowUpContentEphemeral(s, i, content)
	}
}

func ReplyWithError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot encountered a non-fatal error during this command.\n" + errStr

	// Not already deferred, reply.
	_, err = discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		discordutil.FollowUpContentEphemeral(s, i, content)
	}
}
