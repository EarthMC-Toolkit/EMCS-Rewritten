package discordutil

import "github.com/bwmarrin/discordgo"

func OpenModal(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: data,
	})
}

func Reply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// Responds to an interaction with a deferred response, allowing more time to process before sending a follow-up message.
func DeferReply(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
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
