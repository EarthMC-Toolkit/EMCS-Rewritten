package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/database/store"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
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
	//discordutil.ReplyWithError(s, i.Interaction, errors.New("Command not implemented yet."))

	msg := discordutil.NewMessageBuilder()
	msg.SetContent(shared.EMOJIS.LOADING + " This command may take a short while, please wait...")
	if err := discordutil.SendReply(s, i.Interaction, msg.InteractionData()); err != nil {
		return err
	}

	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	_, err = sendFallingTowns(s, i.Interaction, townStore)
	return err
}

func sendFallingTowns(
	s *discordgo.Session, i *discordgo.Interaction,
	townStore *store.Store[oapi.TownInfo],
) (*discordgo.Message, error) {
	// Get all mayors and create a lookup map to their town.
	mayorTownLookup := townStore.EntriesFunc(func(t oapi.TownInfo) string {
		return t.Mayor.UUID
	})
	mayorIds := lo.Keys(mayorTownLookup)
	mayors, err := oapi.QueryPlayers(mayorIds...).Execute()
	if err != nil {
		return nil, err
	}

	// Remove players that are active and not near falling.
	mayors = filterActiveMayors(mayors, time.Now())
	sortLeastActive(mayors)

	// Get the towns of the remaining mayors
	towns := lo.Map(mayors, func(m oapi.PlayerInfo, _ int) oapi.TownInfo {
		return mayorTownLookup[m.UUID]
	})
	sortLowestRes(towns) // Sort towns by amount of residents, lowest first.

	// Output paginator

	return nil, nil
}

// Filters out any mayors not within INACTIVE_THRESHOLD->FALL_DAYS of inactivity.
func filterActiveMayors(mayors []oapi.PlayerInfo, now time.Time) []oapi.PlayerInfo {
	return lo.Filter(mayors, func(p oapi.PlayerInfo, _ int) bool {
		if p.Timestamps.LastOnline == nil {
			return false // must be an NPC (already ruined town). also filter them out
		}

		lo := int64(*p.Timestamps.LastOnline)
		inactive := now.Sub(time.Unix(lo, 0))

		day := 24 * time.Hour
		return inactive >= INACTIVE_THRESHOLD*day && inactive <= 42*day
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

func sortLowestRes(towns []oapi.TownInfo) {
	utils.KeySort(towns, []utils.KeySortOption[oapi.TownInfo]{{
		Compare: func(a, b oapi.TownInfo) bool {
			return a.NumResidents() < b.NumResidents()
		},
	}})
}
