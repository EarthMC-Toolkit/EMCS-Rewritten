package discordutil

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func OpenModal(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: data,
	})
}

func DeferComponent(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

// Responds to an interaction with a deferred response, allowing more time to process before sending a follow-up message.
//
// Deferred interactions cannot carry data and can only be edited or followed up.
func DeferReply(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func SendReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func EditReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
	return s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:         &data.Content,
		Embeds:          &data.Embeds,
		Files:           data.Files,
		Components:      &data.Components,
		AllowedMentions: data.AllowedMentions,
	})
}

func EditOrSendReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
	if err := SendReply(s, i, data); err == nil {
		return nil, nil
	}

	return EditReply(s, i, data)
}

// func SendOrFollowup(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData, ephemeral bool) (*discordgo.Message, error) {
// 	// Try sending the initial reply
// 	err := s.InteractionRespond(i, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: data,
// 	})
// 	if err == nil {
// 		return nil, nil // initial reply sent successfully
// 	}

// 	// If it failed because a response already exists, send a follow-up
// 	return s.FollowupMessageCreate(i, ephemeral, &discordgo.WebhookParams{
// 		Content:         data.Content,
// 		Embeds:          data.Embeds,
// 		Files:           data.Files,
// 		Components:      data.Components,
// 		AllowedMentions: data.AllowedMentions,
// 	})
// }

// func EditOrSendReplyFetch(s *discordgo.Session, i *discordgo.Interaction, fetch bool, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
// 	if err := SendReply(s, i, data); err == nil {
// 		if !fetch {
// 			return nil, nil
// 		}

// 		msg, err := s.InteractionResponse(i)
// 		if err != nil {
// 			return nil, err
// 		}

// 		return msg, nil
// 	}

// 	msg, err := EditReply(s, i, data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return msg, nil
// }

func ReplyWithError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot encountered a non-fatal error during this command.\n" + errStr

	// Not already deferred, reply.
	_, err = EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		FollowupContentEphemeral(s, i, content)
	}
}

func ReplyWithPanicError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot attempted to fatally crash during this command! Please report the following error.\n" + errStr

	// Not already deferred, reply.
	_, err = EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		FollowupContentEphemeral(s, i, content)
	}
}

// Creates a follow-up message for a previously deferred interaction response.
// This func waits for server confirmation of message send and ensures that the return struct is populated.
func Followup(s *discordgo.Session, i *discordgo.Interaction, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return s.FollowupMessageCreate(i, true, params)
}

// Calls FollowUp with the supplied embeds.
func FollowupEmbeds(s *discordgo.Session, i *discordgo.Interaction, embeds ...*discordgo.MessageEmbed) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Embeds: embeds,
	})
}

// Calls FollowUp with the supplied content.
func FollowupContent(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Content: content,
	})
}

// Calls FollowUp with the supplied content which will only be visible to the interaction author.
func FollowupContentEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

// Attempts to get the username from an interaction.
//
// Regular `User` is only filled for a DM, so this func uses guild-specific `Member.User` otherwise.
func GetInteractionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.User != nil {
		return i.User
	}

	return i.Member.User
}

func GetModalInputs(i *discordgo.Interaction) map[string]string {
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
