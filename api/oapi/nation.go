package oapi

import (
	"emcsrw/utils"
	"slices"
	"strings"

	"github.com/samber/lo"
)

type NationStatus struct {
	Public  bool `json:"isPublic"`
	Open    bool `json:"isOpen"`
	Neutral bool `json:"isNeutral"`
}

type NationStats struct {
	NumTownBlocks int     `json:"numTownBlocks"`
	NumResidents  int     `json:"numResidents"`
	NumTowns      int     `json:"numTowns"`
	NumAllies     int     `json:"numAllies"`
	NumEnemies    int     `json:"numEnemies"`
	Balance       float32 `json:"balance"`
}

type NationRanks struct {
	Chancellor []Entity `json:"Chancellor,omitempty"`
	Colonist   []Entity `json:"Colonist,omitempty"`
	Diplomat   []Entity `json:"Diplomat,omitempty"`
}

type NationTimestamps struct {
	Timestamps
}

type NationInfo struct {
	Entity
	King             Entity              `json:"king"`
	Board            string              `json:"board"`
	Wiki             *string             `json:"wiki"`
	MapColourFill    string              `json:"dynmapColour"`
	MapColourOutline string              `json:"dynmapOutline"`
	Capital          Entity              `json:"capital"`
	Residents        []Entity            `json:"residents"`
	Towns            []Entity            `json:"towns"`
	Allies           []Entity            `json:"allies"`
	Enemies          []Entity            `json:"enemies"`
	Sanctioned       []Entity            `json:"sanctioned"`
	Ranks            map[string][]Entity `json:"ranks"`
	Timestamps       NationTimestamps    `json:"timestamps"`
	Status           NationStatus        `json:"status"`
	Stats            NationStats         `json:"stats"`
	Coordinates      struct {
		Spawn Spawn `json:"spawn"`
	} `json:"coordinates"`
}

func (n NationInfo) GetPlayerRanks(name string) []string {
	return lo.FilterMapToSlice(n.Ranks, func(rank string, entities []Entity) (string, bool) {
		return rank, slices.ContainsFunc(entities, func(e Entity) bool {
			return strings.EqualFold(name, e.Name)
		})
	})
}

func (n NationInfo) GetUUID() string {
	return n.UUID
}

func (n NationInfo) Bal() float32 {
	return n.Stats.Balance
}

// Returns the fill colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) FillColourInt() int {
	return utils.HexToInt(n.MapColourFill)
}

// Returns the outline colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) OutlineColourInt() int {
	return utils.HexToInt(n.MapColourOutline)
}
