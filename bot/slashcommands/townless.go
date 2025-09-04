package slashcommands

import (
	"emcsrw/api/mapi"
	"emcsrw/bot/discordutil"

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
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	visible, err := mapi.GetVisiblePlayers()
	if err != nil {
		_, err := discordutil.FollowUpContent(s, i.Interaction, "An error occurred during the map request or response parsing :(")
		return err
	}

	if len(visible) == 0 {
		_, err := discordutil.FollowUpContent(s, i.Interaction, "An error occurred. Players array is empty (server may be partially down).")
		return err
	}

	// Create a paginator
	// paginator := discordutil.NewInteractionPaginator(s, i.ChannelID, i.Member.User.ID, func(page int, data *discordgo.InteractionResponseData) {

	// })

	return nil
}

// func SwitchTownlessPage() {
// 	names := lop.Map(visible, func(p mapi.MapPlayer, i int) string {
// 		return p.Name
// 	})

// 	players, err := oapi.QueryPlayers(names...)
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
