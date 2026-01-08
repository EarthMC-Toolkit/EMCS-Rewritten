package database

import (
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"slices"
)

var DEFAULT_NATION_WEIGHTS = NationStats{
	Residents: 10.0,
	Towns:     6.0,
	Worth:     0.1,
}

type NationWeights NationStats
type NationStats struct {
	Residents, Towns, Worth float64
}

type RankedNation struct {
	UUID  string
	Score float64
	Stats NationStats
	Rank  int
}

func GetRankedNations(
	nationStore *store.Store[oapi.NationInfo],
	w NationWeights,
) map[string]RankedNation {
	nations := nationStore.Values()
	count := len(nations)

	totalResidents, totalTowns, totalWorth := 0, 0, 0
	for _, n := range nations {
		totalResidents += n.NumResidents()
		totalTowns += n.NumTowns()
		totalWorth += n.Worth()
	}

	// compute stats and weighted score
	rankedSlice := make([]RankedNation, count)
	for i, n := range nations {
		stats := NationStats{
			Residents: float64(n.NumResidents()),
			Towns:     float64(n.NumTowns()),
			Worth:     float64(n.Worth()),
		}

		score := (stats.Residents/float64(totalResidents))*w.Residents +
			(stats.Towns/float64(totalTowns))*w.Towns +
			(stats.Worth/float64(totalWorth))*w.Worth

		rankedSlice[i] = RankedNation{
			UUID:  n.UUID,
			Score: score,
			Stats: stats,
		}
	}

	// sort descending by score and assign rank
	slices.SortFunc(rankedSlice, func(a, b RankedNation) int {
		return cmp.Compare(a.Score, b.Score)
	})
	for i := range rankedSlice {
		rankedSlice[i].Rank = i + 1
	}

	out := make(map[string]RankedNation, count)
	for _, rn := range rankedSlice {
		out[rn.UUID] = rn
	}

	return out
}
