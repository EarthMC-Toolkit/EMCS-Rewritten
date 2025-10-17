package slashcommands

import "github.com/bwmarrin/discordgo"

type NewDayCommand struct{}

func (cmd NewDayCommand) Name() string { return "example" }
func (cmd NewDayCommand) Description() string {
	return "This is an example description for a slash command."
}

func (cmd NewDayCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Name:        "when",
			Description: "Sends the time at which the elusive new day will occur.",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
	}
}

func (cmd NewDayCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("when"); opt != nil {
		return executeNewDayWhen(s, i.Interaction)
	}

	return nil
}

func executeNewDayWhen(s *discordgo.Session, i *discordgo.Interaction) error {
	return nil
}
