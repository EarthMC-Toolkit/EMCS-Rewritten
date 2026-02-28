package capi

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"slices"
	"sort"
	"sync"

	"github.com/samber/lo"
)

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders     []string         `json:"leaders"`
	ImageURL    *string          `json:"imageURL"`
	DiscordCode *string          `json:"discordCode"`
	Colours     *AllianceColours `json:"colours"`
}

type AllianceStats struct {
	NumPuppetAlliances int `json:"numPuppetAlliances"`
	NumPuppetNations   int `json:"numPuppetNations"`
	NumResidents       int `json:"numResidents"`
	NumTowns           int `json:"numTowns"`
	NumTownBlocks      int `json:"numTownBlocks"`
	Worth              int `json:"worth"`
}

// type PuppetMap map[string][]string // Key is puppet alliance identifier, value is all it's nation names.
type Alliance struct {
	UUID             uint64            `json:"uuid"`                     // First 48bits = ms timestamp. Extra 16bits = randomness.
	Identifier       string            `json:"identifier"`               // Case-insensitive colloquial short name for lookup.
	Label            string            `json:"label"`                    // Full name for display purposes.
	RepresentativeID *string           `json:"representativeID"`         // Discord ID of the user representing this alliance.
	Parent           *string           `json:"parentAlliance,omitempty"` // The Identifier (not UUID) of the parent alliance this alliance is a child of.
	OwnNations       []string          `json:"ownNations"`               // Names of nations in THIS alliance only.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"`         // Unix timestamp (ms) denoting the time of the last update made to this alliance.
	CreatedTimestamp uint64            `json:"createdTimestamp"`         // Unix timestamp (ms) denoting the time at which this alliance was created.
	Optional         AllianceOptionals `json:"optional"`                 // Extra properties that are not required for basic alliance functionality.
	Type             string            `json:"type"`                     // Type of alliance. See AllianceType consts.
	Stats            AllianceStats     `json:"stats"`                    // Information about ranks within the alliance.
}

func parseAlliance(
	a database.Alliance, alliances []database.Alliance,
	nationStore *store.Store[oapi.NationInfo],
	reslist, townlesslist *oapi.EntityList,
) Alliance {
	ownNations := nationStore.GetFromSet(a.OwnNations)
	ownNationNames := lo.Map(ownNations, func(n oapi.NationInfo, _ int) string {
		return n.Name
	})

	puppetAlliances := a.ChildAlliances(alliances)
	puppetNations := nationStore.GetFromSet(puppetAlliances.NationIds())

	towns, residents, area, worth := a.GetStats(ownNations, puppetNations)
	return Alliance{
		UUID:             a.UUID,
		Identifier:       a.Identifier,
		Label:            a.Label,
		RepresentativeID: a.RepresentativeID,
		Parent:           a.Parent,
		OwnNations:       ownNationNames,
		Type:             string(a.Type),
		UpdatedTimestamp: a.UpdatedTimestamp,
		CreatedTimestamp: a.CreatedTimestamp(),
		Optional: AllianceOptionals{
			Leaders:     a.GetLeaderNames(reslist, townlesslist),
			ImageURL:    a.Optional.ImageURL,
			DiscordCode: a.Optional.DiscordCode,
			Colours:     (*AllianceColours)(a.Optional.Colours),
		},
		Stats: AllianceStats{
			NumPuppetAlliances: len(puppetAlliances),
			NumPuppetNations:   len(puppetNations),
			NumResidents:       residents,
			NumTowns:           len(towns),
			NumTownBlocks:      area,
			Worth:              worth,
		},
	}
}

// Parses input slice of [database.Alliance] to [Alliance] in parallel and returns
// the slice in alphabetical order according to the alliance Identifier.
func getParsedAlliances(
	alliances []database.Alliance, nationStore *store.Store[oapi.NationInfo],
	reslist, townlesslist *oapi.EntityList,
) []Alliance {
	parsedAlliances := make([]Alliance, len(alliances))

	var wg sync.WaitGroup
	wg.Add(len(alliances))

	for i, alliance := range alliances {
		a := alliance // capture loop variable
		idx := i      // capture index
		go func() {
			defer wg.Done()
			parsedAlliances[idx] = parseAlliance(a, alliances, nationStore, reslist, townlesslist)
		}()
	}

	wg.Wait()

	for _, a := range parsedAlliances {
		slices.Sort(a.OwnNations)
		slices.Sort(a.Optional.Leaders)
	}

	// Persist alphabetical order through requests.
	sort.Slice(parsedAlliances, func(i, j int) bool {
		return parsedAlliances[i].Identifier < parsedAlliances[j].Identifier
	})

	return parsedAlliances
}
