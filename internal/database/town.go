package database

import (
	"emcsrw/pkg/api/oapi"
	"time"

	"github.com/samber/lo"
)

const FALL_DAYS = 42     // Total days a mayor can be inactive until their town falls.
const FALLING_TFRAME = 7 // Time frame in days at which to include falling towns.

type MayorUUID = string
type FallingTown struct {
	oapi.TownInfo   // TODO: Maybe just replace this with the town UUID and use TOWNS_STORE to retrieve info
	MayorLastOnline time.Time
	RuinAt          time.Time
	DeletionAt      time.Time
	DaysInactive    float64
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
			TownInfo:        town,
			MayorLastOnline: lo,
			RuinAt:          ruinAt,
			DeletionAt:      deletionAt,
			DaysInactive:    now.Sub(lo).Hours() / 24,
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
