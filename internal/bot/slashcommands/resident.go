package slashcommands

import (
	"emcsrw/pkg/utils/discordutil"

	"github.com/bwmarrin/discordgo"
)

type ResidentCommand struct{}

func (cmd ResidentCommand) Name() string { return "res" }
func (cmd ResidentCommand) Description() string {
	return "Replies with information about a player. Alias of /player query"
}

func (cmd ResidentCommand) Options() []AppCommandOpt {
	return []AppCommandOpt{
		discordutil.RequiredStringOption("name", "The name of the player to query.", 3, 36),
	}
}

func (cmd ResidentCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()
	playerNameArg := cdata.GetOption("name").StringValue()

	// executeQueryPlayer handles its own defer/reply logic
	_, err := executeQueryPlayer(s, i.Interaction, playerNameArg)
	return err
}
