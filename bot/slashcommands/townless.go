package slashcommands

import (
	"emcsrw/utils/discordutil"

	"github.com/bwmarrin/discordgo"
)

type TownlessCommand struct{}

func (cmd TownlessCommand) Name() string { return "townless" }
func (cmd TownlessCommand) Description() string {
	return "Retrieve information on townless players."
}

func (cmd TownlessCommand) Options() AppCommandOpts {
	return nil
}

func (cmd TownlessCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	return nil
}
