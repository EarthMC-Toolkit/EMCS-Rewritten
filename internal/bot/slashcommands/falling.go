package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

// TODO: Add a rate limiter to this command to stop the token bucket being empty.
// Keep a record of time last used per user, and allow again after X minutes.
type FallingCommand struct {
	//lastUsed map[string]int64
}

func (cmd FallingCommand) Name() string { return "falling" }
func (cmd FallingCommand) Description() string {
	return "Sends a list of towns about to fall into ruin within 7 days. Includes stats, status etc."
}

func (cmd FallingCommand) Options() []AppCommandOpt {
	return nil
}

func (cmd FallingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	msg := discordutil.NewMessageBuilder()
	msg.SetContent(shared.EMOJIS.LOADING + " This command may take a short while, please wait...")
	if err := discordutil.SendReply(s, i.Interaction, msg.InteractionData()); err != nil {
		return err
	}

	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	fallingTownsStore, err := database.GetStore(mdb, database.FALLING_TOWNS_STORE)
	if err != nil {
		return err
	}

	falling := fallingTownsStore.Values()
	sort.Slice(falling, func(i, j int) bool {
		if falling[i].DaysInactive == falling[j].DaysInactive {
			return falling[i].TownInfo.NumResidents() < falling[j].TownInfo.NumResidents()
		}
		return falling[i].DaysInactive > falling[j].DaysInactive
	})

	totalCount := len(falling)

	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, totalCount, perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(totalCount)
		items := falling[start:end]

		descBuilder := strings.Builder{} // More performant than concat via regular Sprintf
		for idx, t := range items {
			nationName := "No Nation"
			if t.Nation.Name != nil {
				nationName = *t.Nation.Name
			}

			X, Y, Z := t.SpawnLocation()
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=6)", X, Y, Z, X, Z)

			residents := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.RESIDENT_PURPLE, t.NumResidents())
			balance := logutil.HumanizedSprintf("%s `%.0f` ", shared.EMOJIS.GOLD_INGOT, t.Bal())
			chunks := logutil.HumanizedSprintf("%s `%d` ", shared.EMOJIS.CHUNK, t.Size())

			emojis := shared.EMOJIS
			capital := fmt.Sprintf("%s Capital", lo.Ternary(t.Status.Capital, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			open := fmt.Sprintf("%s Open", lo.Ternary(t.Status.Open, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			spawn := fmt.Sprintf("%s Outsider Spawn", lo.Ternary(t.Status.CanOutsidersSpawn, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			pvp := fmt.Sprintf("%s PVP", lo.Ternary(t.Perms.Flags.PVP, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))

			fmt.Fprintf(&descBuilder, "%d. **%s** (%s) will fall <t:%d:R> at %s.\n"+
				"Deletion on `%s` (<t:%d:R>).\n"+
				"Mayor: `%s` (online <t:%d:R>). %s • %s • %s\n"+
				"%s %s %s %s\n\n",
				start+idx+1, t.Name, nationName, t.RuinAt.Unix(), locationLink, // line 1
				utils.FormatTime(t.DeletionAt), t.DeletionAt.Unix(), // line 2
				t.Mayor.Name, t.MayorLastOnline.Unix(), residents, balance, chunks, // line 3
				capital, open, spawn, pvp, // line 4
			)
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		title := fmt.Sprintf("[%d] List of Falling Towns | %s", totalCount, pageStr)
		desc := descBuilder.String()

		embed := discordutil.NewEmbedBuilder(&discordutil.YELLOW, &title, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}
