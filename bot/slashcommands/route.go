package slashcommands

import "github.com/bwmarrin/discordgo"

// TODO: Implement
type RouteCommand struct{}

func (cmd RouteCommand) Name() string { return "route" }
func (cmd RouteCommand) Description() string {
	return "Get the optimal 'route' to a given point. Includes distance, direction and closest spawn."
}

func (cmd RouteCommand) Options() AppCommandOpts {
	return nil
}

func (cmd RouteCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}
