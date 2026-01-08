package oapi

import (
	"slices"
	"strconv"

	"github.com/samber/lo/parallel"
)

type TownStatus struct {
	Public             bool `json:"isPublic"`
	Open               bool `json:"isOpen"`
	Neutral            bool `json:"isNeutral"`
	Capital            bool `json:"isCapital"`
	Overclaimed        bool `json:"isOverClaimed"`
	Ruined             bool `json:"isRuined"`
	ForSale            bool `json:"isForSale"`
	HasNation          bool `json:"hasNation"`
	HasOverclaimShield bool `json:"hasOverclaimShield"`
	CanOutsidersSpawn  bool `json:"canOutsidersSpawn"`
}

type TownStats struct {
	NumTownBlocks uint32   `json:"numTownBlocks"`
	MaxTownBlocks uint32   `json:"maxTownBlocks"`
	BonusBlocks   uint16   `json:"bonusBlocks"`
	NumResidents  uint32   `json:"numResidents"`
	NumTrusted    uint16   `json:"numTrusted"`
	NumOutlaws    uint16   `json:"numOutlaws"`
	Balance       float32  `json:"balance"`
	ForSalePrice  *float32 `json:"forSalePrice"`
}

type TownCoords struct {
	Spawn      Spawn   `json:"spawn"`
	HomeBlock  [2]int  `json:"homeBlock"`
	TownBlocks [][]int `json:"townBlocks"`
}

type TownRanks struct {
	Councillor []Entity `json:"Councillor,omitempty"`
	Builder    []Entity `json:"Builder,omitempty"`
	Recruiter  []Entity `json:"Recruiter,omitempty"`
	Police     []Entity `json:"Police,omitempty"`
	TaxExempt  []Entity `json:"Tax-exempt,omitempty"`
	Treasurer  []Entity `json:"Treasurer,omitempty"`
}

type TownTimestamps struct {
	Timestamps
	JoinedNationAt *uint64 `json:"joinedNationAt"`
	RuinedAt       *uint64 `json:"ruinedAt"`
}

type TownInfo struct {
	Entity
	Board       string               `json:"board"` // Could be nil, but we want it to default to zero value anyway.
	Wiki        string               `json:"wiki"`  // Could be nil, but we want it to default to zero value anyway.
	Founder     string               `json:"founder"`
	Mayor       Entity               `json:"mayor"`
	Nation      EntityNullableValues `json:"nation"`
	Residents   []Entity             `json:"residents"`
	Timestamps  TownTimestamps       `json:"timestamps"`
	Status      TownStatus           `json:"status"`
	Stats       TownStats            `json:"stats"`
	Coordinates TownCoords           `json:"coordinates"`
	Ranks       TownRanks            `json:"ranks"`
	Trusted     []Entity             `json:"trusted"`
	Outlaws     []Entity             `json:"outlaws"`
	Perms       Perms                `json:"perms"`
	Quarters    []Entity             `json:"quarters"`
}

func (t TownInfo) GetUUID() string {
	return t.UUID
}

func (t TownInfo) Bal() float32 {
	return t.Stats.Balance
}

func (t TownInfo) NumResidents() uint32 {
	return t.Stats.NumResidents
}

func (t TownInfo) Size() uint32 {
	return t.Stats.NumTownBlocks
}

func (t TownInfo) MaxSize() uint32 {
	return t.Stats.MaxTownBlocks
}

// Calculate the worth of the town based on its size, where the first chunk is 64G
// (when the town is created) and each additional chunk is 16G.
func (t TownInfo) Worth() uint32 {
	initialCost := uint32(64)
	if t.Size() <= 1 {
		return initialCost // Shouldn't rly have a town with 0 chunks, but here just in case.
	}

	chunkCost := uint32(16)
	extra := t.Size() - 1

	return initialCost + extra*chunkCost
}

// Unlike Worth(), this func takes everything into account and provides a breakdown
// of the towns total wealth including bank balances in addition to chunks.
func (t TownInfo) Wealth() float32 {
	return float32(t.Worth()) + t.Bal()
}

func (t TownInfo) OverclaimedString() string {
	return strconv.FormatBool(t.Status.Overclaimed)
}

func (t TownInfo) GetResidentNames(alphabetical bool) []string {
	names := parallel.Map(t.Residents, func(t Entity, _ int) string {
		return t.Name
	})

	if alphabetical {
		slices.Sort(names)
	}

	return names
}

func (t TownInfo) GetOnlineResidents() ([]Entity, error) {
	online, err := QueryList(ENDPOINT_ONLINE)
	if err != nil {
		return nil, err
	}

	residentUUIDs := make(map[string]struct{}, len(t.Residents))
	for _, r := range t.Residents {
		residentUUIDs[r.UUID] = struct{}{}
	}

	idx := 0
	for _, op := range online {
		if _, ok := residentUUIDs[op.UUID]; ok {
			online[idx] = op
			idx++
		}
	}

	return online[:idx], nil
}

// type Resident struct {
// 	Entity
// 	Town TownInfo
// }

// // Returns a an instance of [Resident] if it exists within this town, otherwise nil.
// func (t TownInfo) GetResidentByName(name string) (res *Resident) {
// 	parallel.ForEach(t.Residents, func(e Entity, _ int) {
// 		if name == e.Name {
// 			res = &Resident{Entity: e, Town: t} // associate the resident with this town
// 		}
// 	})

// 	return
// }
