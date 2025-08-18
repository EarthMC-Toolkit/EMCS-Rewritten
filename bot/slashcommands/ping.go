package slashcommands

import "github.com/bwmarrin/discordgo"

type Ping struct{}

func (Ping) Name() string                           { return "ping" }
func (Ping) Description() string                    { return "Replies with pong" }
func (Ping) Type() discordgo.ApplicationCommandType { return discordgo.ChatApplicationCommand }
func (Ping) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{}
}

func (Ping) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong!",
		},
	})
}

func init() {
	Register(Ping{})
}
