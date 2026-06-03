package oapi

import (
	"emcsrw/utils"
	"emcsrw/utils/sets"
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
	NationBonus   int     `json:"nationBonus"`
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

type NationPact struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Status   string `json:"status"` // ACTIVE, PENDING (guessing again bc veyronity is a fat lazy retard)
	Stats    struct {
		CreatedAt int64 `json:"createdAt"`
		ExpiresAt int64 `json:"expiresAt"`
		Duration  int   `json:"duration"`
	} `json:"stats"`
}

type NationPacts struct {
	Active  []NationPact `json:"active"`
	Pending []NationPact `json:"pending"`
}

type NationEmbargoes struct {
	Own     []Entity `json:"own"`     // All nations that this nation has placed an embargo on
	Against []Entity `json:"against"` // All nations that this nation has placed an embargo on
}

type NationInfo struct {
	Entity
	MapColourFill    string              `json:"dynmapColour"`
	MapColourOutline string              `json:"dynmapOutline"`
	Board            string              `json:"board"` // Could be nil in response, but default to zero value anyway.
	Wiki             string              `json:"wiki"`  // Could be nil in response, but default to zero value anyway.
	King             Entity              `json:"king"`
	Discord          *string             `json:"discord"`
	Capital          Entity              `json:"capital"`
	Residents        []Entity            `json:"residents"`
	Towns            []Entity            `json:"towns"`
	Allies           []Entity            `json:"allies"`
	Enemies          []Entity            `json:"enemies"`
	Sanctioned       []Entity            `json:"sanctioned"`
	Ranks            map[string][]Entity `json:"ranks"`
	Embargoes        NationEmbargoes     `json:"embargoes"`
	Pacts            NationPacts         `json:"pacts"`
	Timestamps       NationTimestamps    `json:"timestamps"`
	Status           NationStatus        `json:"status"`
	Stats            NationStats         `json:"stats"`
	Coordinates      struct {
		Spawn Spawn `json:"spawn"`
	} `json:"coordinates"`
}

// Gets all ranks for a specified player.
// This func will go through all possible ranks and build a slice of the ranks they hold.
//
// For example, with the name input "Fix" and the nation ranks are as follows:
//
//	{
//	  "Chancellor": ["Fix"],
//	  "Colonist": [],
//	  "Diplomat": ["Fix"]
//	}
//
// The output would be:
//
//	[]string{"Chancellor", "Diplomat"}
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

func (n NationInfo) Spawn() Spawn {
	return n.Coordinates.Spawn
}

// Returns the fill colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) FillColourInt() int {
	return utils.HexToInt(n.MapColourFill)
}

// Returns the outline colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) OutlineColourInt() int {
	return utils.HexToInt(n.MapColourOutline)
}

func (n NationInfo) NumResidents() int {
	return n.Stats.NumResidents
}

func (n NationInfo) NumTowns() int {
	return n.Stats.NumTowns
}

func (n NationInfo) Size() int {
	return n.Stats.NumTownBlocks
}

func (n NationInfo) Worth() int {
	numTowns := n.Stats.NumTowns
	base := numTowns * 64               // every towns first initial chunk (town creation cost)
	extra := (n.Size() - numTowns) * 16 // remaining claimed chunks

	return base + extra
}

func (n NationInfo) GetOnlineResidents() ([]Entity, error) {
	res, err := QueryOnline().Execute()
	if err != nil {
		return nil, err
	}

	residentUUIDs := sets.Make[string](len(n.Residents))
	for _, r := range n.Residents {
		residentUUIDs.Append(r.UUID)
	}

	filtered := res.Players[:0]
	for _, op := range res.Players {
		if residentUUIDs.Has(op.UUID) {
			filtered = append(filtered, op)
		}
	}

	return filtered, nil
}
