package slashcommands

import (
	"emcsrw/utils/discordutil"

	"github.com/bwmarrin/discordgo"
)

type ResidentCommand struct{}

func (cmd ResidentCommand) Name() string { return "res" }
func (cmd ResidentCommand) Description() string {
	return "Replies with information about a player. Alias of /player query"
}

func (cmd ResidentCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		discordutil.RequiredStringOption("name", "The name of the player to query.", 3, 36),
	}
}

func (cmd ResidentCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	playerNameArg := cdata.GetOption("name").StringValue()

	_, err = executeQueryPlayer(s, i.Interaction, playerNameArg)
	return err
}
