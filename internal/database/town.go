package database

import (
	"emcsrw/internal/database/store"
	"emcsrw/pkg/api/oapi"
	"sort"
	"time"

	"github.com/samber/lo"
)

const FALL_DAYS = 42     // Total days a mayor can be inactive until their town falls.
const FALLING_TFRAME = 7 // Time frame in days at which to include falling towns.

type MayorUUID = string
type FallingTown struct {
	oapi.TownInfo                  // TODO: Maybe just replace this with the town UUID and use TOWNS_STORE to retrieve info
	RuinAt           time.Time     // The time at which this town will fall into ruin (next new day 42d after MayorLastOnline).
	DeletionAt       time.Time     // The time at which this falling town will be deleted (next new day 3d after ruin).
	MayorLastOnline  time.Time     // The time at which the mayor last came online.
	InactiveDuration time.Duration // Duration in minutes the mayor has been inactive.
}

func ComputeFallingTowns(mdb *Database, window time.Duration) (map[MayorUUID]FallingTown, error) {
	townStore, err := GetStore(mdb, TOWNS_STORE)
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

	fts := make(map[MayorUUID]FallingTown)
	now := time.Now()

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
		fts[town.UUID] = FallingTown{
			TownInfo:         town,
			MayorLastOnline:  lo,
			RuinAt:           ruinAt,
			DeletionAt:       deletionAt,
			InactiveDuration: now.Sub(lo),
		}
	}

	return fts, nil
}

func nextNewDayAfter(t time.Time) time.Time {
	newDay := time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, time.UTC)
	if !newDay.After(t) {
		newDay = newDay.Add(24 * time.Hour)
	}

	return newDay
}

func GetRuinedTowns(townStore *store.Store[oapi.TownInfo]) []oapi.TownInfo {
	towns := townStore.Values()
	ruined := lo.Filter(towns, func(t oapi.TownInfo, _ int) bool {
		return t.Status.Ruined
	})
	sort.Slice(ruined, func(i, j int) bool {
		// Sort by longest time ruined first. A nil panic shouldn't occur since we already filtered out non-ruins.
		return *ruined[i].Timestamps.RuinedAt < *ruined[j].Timestamps.RuinedAt
	})

	return ruined
}
