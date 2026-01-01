package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"emcsrw/utils/sets"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"
)

var DEFAULT_ALLIANCE_WEIGHTS = AllianceWeights{
	Residents: 10,   // Baseline
	Nations:   6,    // 40% less important than Residents
	Towns:     5,    // 50% less important than Residents
	Worth:     0.15, // Worth of all towns in this alliance. Initial cost + town blocks.
}

// UUID -> AllianceRankInfo
type RankedAlliances = map[uint64]AllianceRankInfo
type AllianceRankInfo struct {
	UUID  uint64
	Rank  int
	Score float64
	Stats AllianceStats
}

type AllianceWeights AllianceStats
type AllianceStats struct {
	Residents, Nations, Towns, Worth float64
}

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders     sets.Set[string] `json:"leaders,omitempty"` // All UUIDs of alliance leaders that exist on EMC.
	ImageURL    *string          `json:"imageURL,omitempty"`
	DiscordCode *string          `json:"discordCode,omitempty"`
	Colours     *AllianceColours `json:"colours,omitempty"`
}

type AllianceType string

const (
	AllianceTypeMeganation   AllianceType = "mega"
	AllianceTypeOrganisation AllianceType = "org"
	AllianceTypePact         AllianceType = "pact"
)

func NewAllianceType(s string) AllianceType {
	s = strings.TrimSpace(strings.ToLower(s))

	switch s {
	case "mega", "meganation":
		return AllianceTypeMeganation
	case "org", "organisation", "organization":
		return AllianceTypeOrganisation
	default:
		return AllianceTypePact
	}
}

// The colloquial, long name of the alliance type. Eg: "mega" becomes "Meganation".
func (t AllianceType) Colloquial() string {
	switch t {
	case AllianceTypeMeganation:
		return "Meganation"
	case AllianceTypeOrganisation:
		return "Organisation"
	default:
		return "Pact"
	}
}

func (t AllianceType) Valid() bool {
	switch t {
	case AllianceTypeMeganation, AllianceTypeOrganisation, AllianceTypePact:
		return true
	default:
		return false
	}
}

// Make sure when an alliance is marshalled (like when saving to a file) that the type defaults to pact.
// This way, we avoid the issue where the type can be its zero-value (empty string) when querying.
func (t AllianceType) MarshalJSON() ([]byte, error) {
	if t == "" {
		t = AllianceTypePact
	}

	return json.Marshal(string(t))
}

type Alliance struct {
	UUID             uint64            `json:"uuid"`             // First 48bits = ms timestamp. Extra 16bits = randomness.
	Identifier       string            `json:"identifier"`       // Case-insensitive colloquial short name for lookup.
	Label            string            `json:"label"`            // Full name for display purposes.
	RepresentativeID *string           `json:"representativeID"` // Discord ID of the user representing this alliance.
	Parent           *string           `json:"parent"`           // The Identifier (not UUID) of the parent alliance this alliance is a child of.
	OwnNations       sets.Set[string]  `json:"ownNations"`       // UUIDs of nations in THIS alliance only.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"` // Unix timestamp (ms) at which the last update was made to this alliance.
	Optional         AllianceOptionals `json:"optional"`         // Extra properties that are not required for basic alliance functionality.
	Type             AllianceType      `json:"type"`             // Type of alliance. See AllianceType consts.
}

func (a Alliance) CreatedTimestamp() uint64 {
	return a.UUID >> 16
}

func (a *Alliance) SetUpdated() {
	now := uint64(time.Now().UnixMilli())
	a.UpdatedTimestamp = &now
}

// TODO: Precompute a parent → children map once when loading/updating alliances.
func (a *Alliance) ChildAlliances(alliances []Alliance) (children ChildAlliances) {
	for _, child := range alliances {
		if child.Identifier == a.Identifier || child.Parent == nil {
			continue // skip self and alliances without parent
		}
		if *child.Parent == a.Identifier {
			children = append(children, child)
		}
	}

	return
}

// Attempts to set the alliance leaders given their IGNs.
//
// The leaders are stored in UUID form if they exist, otherwise the IGN will be added to the `invalid` output slice.
func (a *Alliance) SetLeaders(playerStore *store.Store[BasicPlayer], igns ...string) (invalid []string, err error) {
	playerByName := playerStore.EntriesFunc(func(p BasicPlayer) string {
		return strings.ToLower(p.Name)
	})

	// Report names of any leader igns that weren't valid (not found in the player store).
	leaderSet := make(sets.Set[string])
	for _, ign := range igns {
		p, ok := playerByName[strings.ToLower(ign)]
		if !ok {
			invalid = append(invalid, ign)
			continue
		}

		leaderSet.Append(p.UUID)
	}

	a.Optional.Leaders = leaderSet
	return
}

// Returns a map of leaders where key is the leader's UUID and value is their OAPI player data.
// func (a Alliance) QueryLeaders() (map[string]oapi.PlayerInfo, error) {
// 	if len(a.Optional.Leaders) < 1 {
// 		return nil, fmt.Errorf("no leaders exist to query")
// 	}

// 	// I doubt we'll ever have an alliance with more leaders than the player query limit,
// 	// so there shouldn't be a need to send more than one request.
// 	leaders, err := oapi.QueryPlayers(a.Optional.Leaders...)
// 	if err != nil {
// 		return nil, fmt.Errorf("error occurred:\n%v", err)
// 	}

// 	return lo.SliceToMap(leaders, func(p oapi.PlayerInfo) (string, oapi.PlayerInfo) {
// 		return p.UUID, p
// 	}), nil
// }

// Returns a map of leaders where key is the leader's UUID and value is their basic player data.
func (a Alliance) GetLeaders(playerStore *store.Store[BasicPlayer]) ([]BasicPlayer, error) {
	leaderCount := len(a.Optional.Leaders)
	if leaderCount < 1 {
		return nil, fmt.Errorf("cannot query leaders. none set")
	}

	players := playerStore.GetFromSet(a.Optional.Leaders)
	if len(players) < 1 {
		return nil, fmt.Errorf("leader(s) are set but do not exist")
	}

	return players, nil
}

func (a Alliance) GetLeaderNames(reslist, townlesslist *oapi.EntityList) (names []string) {
	for id := range a.Optional.Leaders {
		if name, ok := (*reslist)[id]; ok {
			names = append(names, name)
			continue
		}
		if name, ok := (*townlesslist)[id]; ok {
			names = append(names, name)
		}
	}

	return
}

func (a Alliance) GetStats(ownNations []oapi.NationInfo, childNations []oapi.NationInfo) (
	towns []oapi.Entity,
	residents, area, worth int,
) {
	residents = 0
	area = 0

	lo.ForEach(ownNations, func(n oapi.NationInfo, _ int) {
		residents += n.Stats.NumResidents
		area += n.Stats.NumTownBlocks
		towns = append(towns, n.Towns...)
	})

	lo.ForEach(childNations, func(n oapi.NationInfo, _ int) {
		residents += n.Stats.NumResidents
		area += n.Stats.NumTownBlocks
		towns = append(towns, n.Towns...)
	})

	townsCount := len(towns)

	townsChunkWorth := (area - townsCount) * 16 // Every chunk after the first free one.
	townsBaseWorth := townsCount * 64           // Actual cost of town creation.
	worth = townsChunkWorth + townsBaseWorth

	return
}

func GetRankedAlliances(
	allianceStore *store.Store[Alliance],
	nationStore *store.Store[oapi.NationInfo],
	w AllianceWeights,
) RankedAlliances {
	alliances := allianceStore.Values()

	stats := make([]AllianceStats, len(alliances))
	for i, a := range alliances {
		// collect own nations
		ownNations := nationStore.GetFromSet(a.OwnNations)

		// collect child nations
		childNationIDs := a.ChildAlliances(alliances).NationIds()
		childNations := nationStore.GetFromSet(childNationIDs)

		towns, residents, _, worth := a.GetStats(ownNations, childNations)
		s := AllianceStats{
			Residents: float64(residents),
			Nations:   float64(len(ownNations) + len(childNations)),
			Towns:     float64(len(towns)),
			Worth:     float64(worth),
		}

		stats[i] = s
	}

	ranked := make([]AllianceRankInfo, len(alliances))
	for i, a := range alliances {
		s := stats[i]

		// Scores could be in the millions, scale down to ensure its readable.
		score := s.Residents*w.Residents + s.Towns*w.Towns + s.Nations*w.Nations + s.Worth*w.Worth
		ranked[i] = AllianceRankInfo{
			UUID:  a.UUID,
			Score: score / 4,
			Stats: s,
		}
	}

	// sort by score and assign rank based on new index
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	for i := range ranked {
		ranked[i].Rank = i + 1 // start from 1, where 1 is the best rank.
	}

	out := make(RankedAlliances, len(ranked))
	for _, info := range ranked {
		out[info.UUID] = info
	}

	return out
}

type ChildAlliances []Alliance

func (a ChildAlliances) NationIds() sets.Set[string] {
	s := make(sets.Set[string])
	for _, child := range a {
		for uuid := range child.OwnNations {
			s.Append(uuid)
		}
	}

	return s
}
