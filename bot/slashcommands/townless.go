package slashcommands

import (
	"emcsrw/mapi"

	"github.com/bwmarrin/discordgo"
)

type TownlessCommand struct{}

func (cmd TownlessCommand) Name() string { return "townless" }
func (cmd TownlessCommand) Description() string {
	return "Retrieve a list of online players without a town."
}

func (cmd TownlessCommand) Options() []*discordgo.ApplicationCommandOption {
	return nil
}

func (cmd TownlessCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Defer the interaction immediately
	err := DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	ops, err := mapi.GetOnlinePlayers()
	if err != nil {
		_, err := FollowUpContent(s, i.Interaction, "An error occurred during the map request or response parsing :(")
		return err
	}

	if len(ops) == 0 {
		_, err := FollowUpContent(s, i.Interaction, "An error occurred. Players array is empty (server may be partially down).")
		return err
	}

	// Create a paginator
	//CreatePaginator(s, i.Interaction)

	return nil
}

func CreatePaginator(s *discordgo.Session, i *discordgo.Interaction, switchPage func(page int) error) {

}

// func SwitchTownlessPage() {
// 	opNames := lop.Map(ops, func(op mapi.OnlinePlayer, i int) string {
// 		return op.Name
// 	})

// 	players, err := oapi.QueryPlayers(opNames...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	townless := lo.Filter(players, func(p objs.PlayerInfo, i int) bool {
// 		return !p.Status.HasTown
// 	})
// }

// func SendTownlessList() (*discordgo.Message, error) {
// 	towns, err := oapi.QueryTowns(strings.ToLower(townNameArg))
// 	if err != nil {
// 		return FollowUpContent(s, i, "An error occurred retrieving town information :(")
// 	}

// 	if len(towns) == 0 {
// 		return FollowUpContent(s, i, fmt.Sprintf("No towns retrieved. Town `%s` does not seem to exist.", townNameArg))
// 	}

// 	embed := common.CreateTownEmbed(towns[0])
// 	return FollowUpEmbeds(s, i, embed)
// }
