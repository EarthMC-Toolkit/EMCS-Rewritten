package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

const FALL_DAYS = 42                                  // Total days a mayor can be inactive until their town falls.
const FALLING_TFRAME = 7                              // Time frame in days at which to include falling towns.
const INACTIVE_THRESHOLD = FALL_DAYS - FALLING_TFRAME // Threshold at which a player is considered inactive after.

// TODO: Add a rate limiter to this command to stop the token bucket being empty.
// Keep a record of time last used per user, and allow again after X minutes.
type FallingCommand struct {
	//lastUsed map[string]int64
}

func (cmd FallingCommand) Name() string { return "falling" }
func (cmd FallingCommand) Description() string {
	return "See info about falling towns, their fall date and the closest route to them."
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

	falling, err := GetFallingTowns(mdb)
	if err != nil {
		return err
	}

	totalCount := len(falling)

	perPage := 10
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, totalCount, perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(totalCount)

		items := falling[start:end]
		pageCount := len(items)

		descBuilder := strings.Builder{} // More performant than concat via regular Sprintf
		for idx, t := range items {
			last := time.UnixMilli(t.MayorLastOnline)

			ruinAt := last.Add(FALL_DAYS * 24 * time.Hour)            // RUIN: next new day after threshold
			deletionAt := nextNewDayAfter(ruinAt.Add(72 * time.Hour)) // DELETION: next new day after 3d of ruin

			X, Y, Z := t.SpawnLocation()
			locationLink := fmt.Sprintf(
				"[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=5)",
				X, Y, Z, X, Z,
			)

			chunks := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.CHUNK, t.Size())
			balance := logutil.HumanizedSprintf("%s `%0.0f`", shared.EMOJIS.GOLD_INGOT, t.Bal())

			fmt.Fprintf(&descBuilder, "%d. **%s** will fall into ruin <t:%d:R> at %s. %sG %s\nDeletion on `%s` (<t:%d:R>).\n\n",
				start+idx+1, t.Name, ruinAt.Unix(), locationLink, balance, chunks, // first line
				utils.FormatTime(deletionAt), deletionAt.Unix(), // second line
			)
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		title := fmt.Sprintf("[%d] List of Falling Towns | %s", pageCount, pageStr)
		desc := descBuilder.String()

		embed := discordutil.NewEmbedBuilder(&discordutil.YELLOW, &title, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}

type FallingTown struct {
	oapi.TownInfo
	MayorLastOnline int64
}

func GetFallingTowns(mdb *database.Database) ([]FallingTown, error) {
	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		return nil, err
	}

	// Get all mayors and create a lookup map to their town.
	mayorTownLookup := townStore.EntriesFunc(func(t oapi.TownInfo) string {
		return t.Mayor.UUID
	})
	mayorIds := lo.Keys(mayorTownLookup)
	mayors, err := oapi.QueryPlayers(mayorIds...).Execute()
	if err != nil {
		return nil, err
	}

	// Remove players that are active and not near falling yet.
	mayors = filterActiveMayors(mayors, time.Now())
	sortLeastActive(mayors)

	falling := lo.Map(mayors, func(m oapi.PlayerInfo, _ int) FallingTown {
		return FallingTown{
			TownInfo:        mayorTownLookup[m.UUID],
			MayorLastOnline: int64(*m.Timestamps.LastOnline),
		}
	})

	// Sort towns by least amount of residents first
	utils.KeySort(falling, []utils.KeySortOption[FallingTown]{{
		Compare: func(a, b FallingTown) bool {
			return a.TownInfo.NumResidents() < b.TownInfo.NumResidents()
		},
	}})

	return falling, nil
}

// Filters out any mayors not between INACTIVE_THRESHOLD and FALL_DAYS.
// Aka removes active mayors, keeping the inactive ones who's towns are about to fall.
func filterActiveMayors(mayors []oapi.PlayerInfo, now time.Time) []oapi.PlayerInfo {
	day := 24 * time.Hour
	return lo.Filter(mayors, func(p oapi.PlayerInfo, _ int) bool {
		if p.Timestamps.LastOnline == nil {
			return false // must be an NPC (already ruined town). also filter them out
		}

		ts := int64(*p.Timestamps.LastOnline)
		inactive := now.Sub(time.UnixMilli(ts))

		return inactive >= INACTIVE_THRESHOLD*day && inactive <= FALL_DAYS*day
	})
}

// Sorts a list of players by least active first using their LastOnline timestamp.
func sortLeastActive(players []oapi.PlayerInfo) {
	utils.KeySort(players, []utils.KeySortOption[oapi.PlayerInfo]{{
		Compare: func(a, b oapi.PlayerInfo) bool {
			aLo := a.Timestamps.LastOnline
			bLo := b.Timestamps.LastOnline

			switch {
			case aLo == nil && bLo == nil:
				return false
			case aLo == nil:
				return true // NPCs first
			case bLo == nil:
				return false
			default:
				return *aLo < *bLo
			}
		},
	}})
}

func nextNewDayAfter(t time.Time) time.Time {
	newDay := time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, time.UTC)
	if !newDay.After(t) {
		newDay = newDay.Add(24 * time.Hour)
	}

	return newDay
}
