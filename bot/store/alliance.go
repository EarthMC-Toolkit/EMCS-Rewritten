package store

import (
	"sync"

	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders    *[]string        `json:"leaders,omitempty"`
	ImageURL   *string          `json:"imageURL,omitempty"`
	DiscordURL *string          `json:"discordURL,omitempty"`
	Colours    *AllianceColours `json:"colours,omitempty"`
}

type AllianceType string

const (
	AllianceTypeMeganation   AllianceType = "meganation"
	AllianceTypeOrganisation AllianceType = "organisation"
	AllianceTypePact         AllianceType = "pact"
)

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
