package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

type ExistenceMap = map[string]struct{}

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders     []string         `json:"leaders,omitempty"` // Minecraft UUIDs of alliance leaders.
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
	Parents          []string          `json:"parents"`          // The Identifier (not UUID) of all parents of this alliance.
	OwnNations       []string          `json:"ownNations"`       // UUIDs of nations in THIS alliance only.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"` // Unix timestamp (ms) at which the last update was made to this alliance.
	Optional         AllianceOptionals `json:"optional"`         // Extra properties that are not required for basic alliance functionality.
	Type             AllianceType      `json:"type"`             // Type of alliance. See AllianceType consts.
}

func (a Alliance) CreatedTimestamp() uint64 {
	return a.UUID >> 16
}

func (a *Alliance) GetNations(nationStore *store.Store[oapi.NationInfo]) []oapi.NationInfo {
	return nationStore.GetMany(a.OwnNations...)
}

func (a *Alliance) SetLeaders(igns ...string) (invalid []string, err error) {
	leaders, err := oapi.QueryPlayers(igns...)
	if err != nil {
		return nil, fmt.Errorf("error occurred:\n%v", err)
	}

	found := make(ExistenceMap, len(leaders))
	for _, p := range leaders {
		found[p.Name] = struct{}{}

		// Only add leader if they don't exist already.
		if !slices.Contains(a.Optional.Leaders, p.UUID) {
			a.Optional.Leaders = append(a.Optional.Leaders, p.UUID)
		}
	}

	// Report names of any leader igns that weren't valid.
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

	// I doubt we'll ever have an alliance with more players than the
	// query limit, so there shouldn't be a need to send more than one request.
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

// Returns a map of parent alliance UUID → slice of child alliances.
// For correct results, parameter `alliances` should be a map of all known alliances.
func BuildChildAlliancesMap(alliances map[string]Alliance) map[string][]*Alliance {
	children := make(map[string][]*Alliance)
	mu := sync.Mutex{}

	parallel.ForEach(lo.Values(alliances), func(a Alliance, _ int) {
		for _, parentID := range a.Parents {
			mu.Lock()
			children[parentID] = append(children[parentID], &a)
			mu.Unlock()
		}
	})

	return children
}

// A non-parallel version of BuildChildAlliancesMap in case the parallel one has issues.
// func BuildChildAlliancesMap(alliances map[string]Alliance) map[string][]*Alliance {
// 	children := make(map[string][]*Alliance)
// 	for _, a := range alliances {
// 		for _, parentID := range a.Parents {
// 			children[parentID] = append(children[parentID], &a)
// 		}
// 	}

// 	return children
// }
