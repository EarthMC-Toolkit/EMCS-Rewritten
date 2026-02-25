package slashcommands

import (
	"emcsrw/utils/discordutil"
	"errors"

	"github.com/bwmarrin/discordgo"
)

// TODO: Implement this command
type FallingCommand struct{}

func (cmd FallingCommand) Name() string { return "falling" }
func (cmd FallingCommand) Description() string {
	return "See info about falling towns, their fall date and the closest route to them."
}

func (cmd FallingCommand) Options() AppCommandOpts {
	return nil
}

func (cmd FallingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	discordutil.ReplyWithError(s, i.Interaction, errors.New("Command not implemented yet."))
	return nil
}
