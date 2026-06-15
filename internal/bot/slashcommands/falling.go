package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
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

const FALL_DAYS = 42     // Total days a mayor can be inactive until their town falls.
const FALLING_TFRAME = 7 // Time frame in days at which to include falling towns.

// TODO: Add a rate limiter to this command to stop the token bucket being empty.
// Keep a record of time last used per user, and allow again after X minutes.
type FallingCommand struct {
	//lastUsed map[string]int64
}

func (cmd FallingCommand) Name() string { return "falling" }
func (cmd FallingCommand) Description() string {
	return "See info about falling towns, their fall date, balance etc."
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

	falling, err := ComputeFallingTowns(mdb, time.Now(), FALLING_TFRAME*24*time.Hour)
	if err != nil {
		return err
	}

	totalCount := len(falling)

	perPage := 6
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

			residents := logutil.HumanizedSprintf("`%d`", t.NumResidents())
			balance := logutil.HumanizedSprintf("`%0.0f` %s", t.Bal(), shared.EMOJIS.GOLD_INGOT)
			chunks := logutil.HumanizedSprintf("`%d` %s", t.Size(), shared.EMOJIS.CHUNK)
			fmt.Fprintf(&descBuilder, "%d. **%s** (%s) is scheduled to fall <t:%d:R> at %s. Deletion on `%s` (<t:%d:R>).\n"+
				"Mayor: `%s` (online <t:%d:R>).\n"+
				"Residents: %s Balance: %s Chunks: %s\n\n",
				start+idx+1, t.Name, nationName, t.RuinAt.Unix(), locationLink, utils.FormatTime(t.DeletionAt), t.DeletionAt.Unix(), // line 1
				t.Mayor.Name, t.MayorLastOnline.Unix(), // line 2
				residents, balance, chunks, // line 3
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

type FallingTown struct {
	oapi.TownInfo
	MayorLastOnline time.Time
	RuinAt          time.Time
	DeletionAt      time.Time
	DaysInactive    float64
}

func ComputeFallingTowns(mdb *database.Database, now time.Time, window time.Duration) ([]FallingTown, error) {
	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		return nil, err
	}

	// mayor UUID → town
	mayorTownLookup := townStore.EntriesFunc(func(t oapi.TownInfo) string {
		return t.Mayor.UUID
	})

	mayorIDs := lo.Keys(mayorTownLookup)
	mayors, errs, _ := oapi.QueryPlayers(mayorIDs...).ExecuteConcurrent()
	if len(errs) > 0 {
		return nil, errs[0]
	}

	ft := make([]FallingTown, 0, len(mayors))
	for _, m := range mayors {
		if m.Timestamps.LastOnline == nil {
			continue // NPCs excluded
		}
		town, ok := mayorTownLookup[m.UUID]
		if !ok {
			continue
		}

		lo := time.UnixMilli(int64(*m.Timestamps.LastOnline))
		ruinAt := nextNewDayAfter(lo.Add(FALL_DAYS * 24 * time.Hour))
		timeToRuin := ruinAt.Sub(now) // remaining duration until this town hits its ruin time
		if timeToRuin < 0 || timeToRuin > window {
			continue // remaining duration until this town hits its ruin time
		}

		deletionAt := nextNewDayAfter(ruinAt.Add(72 * time.Hour))
		ft = append(ft, FallingTown{
			TownInfo:        town,
			MayorLastOnline: lo,
			RuinAt:          ruinAt,
			DeletionAt:      deletionAt,
			DaysInactive:    now.Sub(lo).Hours() / 24,
		})
	}

	// Sorts by mayor inactivity duration, then by town size when inactivity is equal.
	sort.Slice(ft, func(i, j int) bool {
		if ft[i].DaysInactive == ft[j].DaysInactive {
			return ft[i].TownInfo.NumResidents() < ft[j].TownInfo.NumResidents()
		}
		return ft[i].DaysInactive > ft[j].DaysInactive
	})

	return ft, nil
}

func nextNewDayAfter(t time.Time) time.Time {
	newDay := time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, time.UTC)
	if !newDay.After(t) {
		newDay = newDay.Add(24 * time.Hour)
	}

	return newDay
}
