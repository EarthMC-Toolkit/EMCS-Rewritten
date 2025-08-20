package slashcommands

import "github.com/bwmarrin/discordgo"

func DeferReply(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func FollowUp(s *discordgo.Session, i *discordgo.Interaction, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return s.FollowupMessageCreate(i, true, params)
}

func FollowUpEmbeds(s *discordgo.Session, i *discordgo.Interaction, embeds ...*discordgo.MessageEmbed) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Embeds: embeds,
	})
}

func FollowUpContent(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return FollowUp(s, i, &discordgo.WebhookParams{
		Content: content,
	})
}
