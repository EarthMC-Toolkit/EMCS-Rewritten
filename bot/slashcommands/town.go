package slashcommands

import "github.com/bwmarrin/discordgo"

type TownCommand struct{}

func (TownCommand) Name() string                           { return "town" }
func (TownCommand) Description() string                    { return "Retrieve information relating to one or more towns." }
func (TownCommand) Type() discordgo.ApplicationCommandType { return discordgo.ChatApplicationCommand }
func (TownCommand) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{}
}

func (TownCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}
