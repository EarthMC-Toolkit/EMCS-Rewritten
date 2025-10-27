package discordutil

import "github.com/bwmarrin/discordgo"

const DEV_ID = "263377802647175170"

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

// Creates a follow-up message for a previously deferred interaction response.
// This func waits for server confirmation of message send and ensures that the return struct is populated.
func FollowUp(s *discordgo.Session, i *discordgo.Interaction, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return s.FollowupMessageCreate(i, true, params)
}

// Calls FollowUp with the supplied embeds.
func FollowUpEmbeds(s *discordgo.Session, i *discordgo.Interaction, embeds ...*discordgo.MessageEmbed) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Embeds: embeds,
	})
}

// Calls FollowUp with the supplied content.
func FollowUpContent(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Content: content,
	})
}

// Calls FollowUp with the supplied content which will only be visible to the interaction author.
func FollowUpContentEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
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

func IsDev(i *discordgo.Interaction) bool {
	author := GetInteractionAuthor(i)
	return author.ID == DEV_ID
}
