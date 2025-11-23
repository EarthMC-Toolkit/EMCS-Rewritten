package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"emcsrw/utils"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
)

type ExistenceMap = map[string]struct{}

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders     []string         `json:"leaders,omitempty"` // All UUIDs of alliance leaders that exist on EMC.
	ImageURL    *string          `json:"imageURL,omitempty"`
	DiscordCode *string          `json:"discordURL,omitempty"`
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

// Make sure when an alliance is marshalled (like when saved to a file), the type defaults to pact.
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
	OwnNations       []string          `json:"ownNations"`       // UUIDs of nations in THIS alliance only.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"` // Unix timestamp (ms) at which the last update was made to this alliance.
	Optional         AllianceOptionals `json:"optional"`         // Extra properties that are not required for basic alliance functionality.
	Type             AllianceType      `json:"type"`             // Type of alliance. See AllianceType consts.
}

func (a Alliance) CreatedTimestamp() uint64 {
	return a.UUID >> 16
}

func (a *Alliance) GetOwnNations(nationStore *store.Store[oapi.NationInfo]) []oapi.NationInfo {
	return nationStore.GetMany(a.OwnNations...)
}

// Returns all nations in this alliance and its child alliances.
//
// If we know beforehand there are no child alliances, prefer GetOwnNations() instead for better performance.
func (a Alliance) GetAllNations(nationStore *store.Store[oapi.NationInfo], allianceStore *store.Store[Alliance]) []oapi.NationInfo {
	allNations := utils.CopySlice(a.OwnNations) // with value receiver, slice is still a reference so a copy is required
	seen := make(ExistenceMap, len(a.OwnNations))
	for _, id := range a.OwnNations {
		seen[id] = struct{}{}
	}

	childAlliances, count := a.GetChildAlliances(allianceStore)
	if count > 0 {
		for _, child := range childAlliances {
			for _, uuid := range child.OwnNations {
				if _, exists := seen[uuid]; exists {
					continue
				}

				seen[uuid] = struct{}{}
				allNations = append(allNations, uuid)
			}
		}
	}

	return nationStore.GetMany(allNations...)
}

// Attempts to set the alliance leaders given their IGNs by querying the Official API.
//
// The leaders are stored in UUID form if they exist, otherwise the IGN will be added to the `invalid` output slice.
func (a *Alliance) SetLeaders(igns ...string) (invalid []string, err error) {
	leaders, err := oapi.QueryPlayers(igns...)
	if err != nil {
		return nil, err
	}

	leaderUUIDs := make([]string, 0, len(leaders))

	found := make(ExistenceMap, len(leaders))
	for _, p := range leaders {
		found[p.Name] = struct{}{}
		leaderUUIDs = append(leaderUUIDs, p.UUID)
	}

	a.Optional.Leaders = leaderUUIDs

	// Report names of any leader igns that weren't valid. I.e, not found in the query results.
	for _, ign := range igns {
		if _, ok := found[ign]; !ok {
			invalid = append(invalid, ign)
		}
	}

	return
}

// Returns a map of leaders where key is the leader's UUID and value is their player data.
func (a Alliance) GetLeaders() (map[string]oapi.PlayerInfo, error) {
	if len(a.Optional.Leaders) < 1 {
		return nil, nil
	}

	// I doubt we'll ever have an alliance with more leaders than the player query limit,
	// so there shouldn't be a need to send more than one request.
	leaders, err := oapi.QueryPlayers(a.Optional.Leaders...) // TODO: Should store import oapi?
	if err != nil {
		return nil, fmt.Errorf("error occurred:\n%v", err)
	}

	return lo.SliceToMap(leaders, func(p oapi.PlayerInfo) (string, oapi.PlayerInfo) {
		return p.UUID, p
	}), nil
}

func (a Alliance) GetLeaderNames() ([]string, error) {
	leaders, err := a.GetLeaders()
	if err != nil {
		return nil, err
	}

	return lo.MapToSlice(leaders, func(_ string, p oapi.PlayerInfo) string {
		return p.Name
	}), nil
}

func (a Alliance) GetStats(
	nationStore *store.Store[oapi.NationInfo],
	allianceStore *store.Store[Alliance],
) (
	nations []oapi.NationInfo, towns []oapi.Entity,
	residents, area int, wealth int,
) {
	nations = a.GetAllNations(nationStore, allianceStore)
	towns = lo.FlatMap(nations, func(n oapi.NationInfo, _ int) []oapi.Entity {
		return n.Towns
	})

	residents = 0
	area = 0
	lo.ForEach(nations, func(n oapi.NationInfo, _ int) {
		residents += n.Stats.NumResidents
		area += n.Stats.NumTownBlocks
	})

	townsCount := len(towns)

	townsChunkWorth := (area - townsCount) * 16 // Every chunk after the first free one.
	townsBaseWorth := townsCount * 64           // Actual cost of town creation.
	wealth = townsChunkWorth + townsBaseWorth

	return
}

// Returns all alliances that are children of this alliance, if any.
func (a Alliance) GetChildAlliances(allianceStore *store.Store[Alliance]) (children []Alliance, count int) {
	parentIdent := a.Identifier
	for _, alliance := range allianceStore.Values() {
		if alliance.Identifier == parentIdent {
			continue // skip self
		}

		if alliance.Parent != nil && *alliance.Parent == parentIdent {
			children = append(children, alliance)
		}
	}

	return children, len(children)
}

// Returns a map of parent alliance UUID → slice of child alliances.
// For correct results, parameter `alliances` should be a map of all known alliances.
// func BuildChildAlliancesMap(alliances []Alliance) map[string][]Alliance {
// 	children := make(map[string][]Alliance)
// 	mu := sync.Mutex{}

// 	parallel.ForEach(alliances, func(a Alliance, _ int) {
// 		if a.Parent == nil {
// 			return
// 		}

// 		mu.Lock()
// 		children[*a.Parent] = append(children[*a.Parent], a)
// 		mu.Unlock()
// 	})

// 	return children
// }

// A non-parallel version of BuildChildAlliancesMap in case the parallel one has issues.
// func BuildChildAlliancesMap(alliances map[string]Alliance) map[string][]*Alliance {
// 	children := make(map[string][]*Alliance)
// 	for _, a := range alliances {
// 		if a.Parent == nil {
// 			continue
// 		}

// 		children[*a.Parent] = append(children[*a.Parent], &a)
// 	}

// 	return children
// }
