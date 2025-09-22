package discordutil

import "github.com/bwmarrin/discordgo"

func OpenModal(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: data,
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

func EditOrSendReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
	msg, err := s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:         &data.Content,
		Embeds:          &data.Embeds,
		Files:           data.Files,
		Components:      &data.Components,
		AllowedMentions: data.AllowedMentions,
	})
	if err == nil {
		return msg, nil
	}

	return nil, SendReply(s, i, data)
}

// Creates a follow-up message for a previously deferred interaction response.
// This func waits for server confirmation of message send and ensures that the return struct is populated.
func FollowUp(s *discordgo.Session, i *discordgo.Interaction, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return s.FollowupMessageCreate(i, true, params)
}

// Calls [FollowUp] with the supplied embeds.
func FollowUpEmbeds(s *discordgo.Session, i *discordgo.Interaction, embeds ...*discordgo.MessageEmbed) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Embeds: embeds,
	})
}

// Calls [FollowUp] with the supplied content.
func FollowUpContent(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Content: content,
	})
}

// Calls [FollowUp] with the supplied content which will only be visible to the interaction author.
func FollowUpContentEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

// Attempts to get the username from an interaction.
//
// Regular `User` is only filled for a DM, so this func uses guild-specific `Member.User` otherwise.
func UserFromInteraction(i *discordgo.Interaction) *discordgo.User {
	if i.User != nil {
		return i.User
	}

	return i.Member.User
}
