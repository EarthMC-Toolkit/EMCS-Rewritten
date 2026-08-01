package oapi

import (
	"emcsrw/pkg/utils/sets"
	"slices"
	"strconv"

	"github.com/samber/lo/parallel"
)

type TownStatus struct {
	Public      bool `json:"isPublic"`
	Open        bool `json:"isOpen"`
	Neutral     bool `json:"isNeutral"`
	Capital     bool `json:"isCapital"`
	Overclaimed bool `json:"isOverClaimed"`
	Ruined      bool `json:"isRuined"`
	ForSale     bool `json:"isForSale"`
	HasNation   bool `json:"hasNation"`
	//HasOverclaimShield bool `json:"hasOverclaimShield"` // removed by sub 100IQ vegetable veyronity
	HasFriendlyFire     bool `json:"hasFriendlyFire"`
	HasSnowAccumulation bool `json:"hasSnowAccumulation"`
	CanOutsidersSpawn   bool `json:"canOutsidersSpawn"`
	CanPassiveMobsSpawn bool `json:"canPassiveMobsSpawn"`
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

type TownWarp struct {
	Entity
	CreatedAt uint64     `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	Access    string     `json:"access"` // RESIDENT, ALLY, NATION, PUBLIC (guessed based on usual towny perms)
	Location  Location3D `json:"location"`
}

type TownInfo struct {
	Entity
	Board       string               `json:"board"` // Could be nil, but we want it to default to zero value anyway.
	Wiki        string               `json:"wiki"`  // Could be nil, but we want it to default to zero value anyway.
	Discord     *string              `json:"discord"`
	Founder     string               `json:"founder"`
	Mayor       Entity               `json:"mayor"`
	Nation      EntityNullableValues `json:"nation"`
	Residents   []Entity             `json:"residents"`
	Timestamps  TownTimestamps       `json:"timestamps"`
	Status      TownStatus           `json:"status"`
	Stats       TownStats            `json:"stats"`
	Coordinates TownCoords           `json:"coordinates"`
	Ranks       TownRanks            `json:"ranks"`
	Perms       Perms                `json:"perms"`
	Trusted     []Entity             `json:"trusted,omitempty"`
	Outlaws     []Entity             `json:"outlaws,omitempty"`
	Quarters    []Entity             `json:"quarters,omitempty"`
	Warps       []TownWarp           `json:"warps,omitempty"`
}

func (t TownInfo) GetName() string {
	return t.Name
}

func (t TownInfo) GetUUID() string {
	return t.UUID
}

func (t TownInfo) GetResidents() []Entity {
	return t.Residents
}

func (t TownInfo) Bal() float32 {
	return t.Stats.Balance
}

func (t TownInfo) Spawn() Spawn {
	return t.Coordinates.Spawn
}

func (t TownInfo) SpawnLocation() (float32, float32, float32) {
	spawn := t.Coordinates.Spawn
	return spawn.X, spawn.Y, spawn.Z
}

func (t TownInfo) IsRuined() bool {
	return t.Status.Ruined
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

// Calculates and returns the value of town's land mass, where the first chunk is 64G
// (when the town is created) and each additional chunk is worth 16G. If the town has 0 chunks, it is worth 0G.
//
// NOTE: This does not related to the towns bank balance in any way, it is purely land value.
func (t TownInfo) LandValue() uint32 {
	if t.Size() <= 1 {
		return uint32(0) // Shouldn't rly have a town with 0 chunks, but here just in case.
	}

	chunkCost := uint32(16)
	extra := t.Size() - 1

	return uint32(64) + (extra * chunkCost)
}

// Returns the total wealth of the town, which is the sum of its land value and bank balance.
// This is a more accurate representation of the town's overall/economic worth.
func (t TownInfo) Wealth() float32 {
	return float32(t.LandValue()) + t.Bal()
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
	res, err := QueryOnline().Execute()
	if err != nil {
		return nil, err
	}

	residentUUIDs := sets.Make[string](len(t.Residents))
	for _, r := range t.Residents {
		residentUUIDs.Add(r.UUID)
	}

	filtered := res.Players[:0]
	for _, op := range res.Players {
		if residentUUIDs.Has(op.UUID) {
			filtered = append(filtered, op)
		}
	}

	return filtered, nil
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
