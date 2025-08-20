package objs

import (
	"slices"
	"strconv"

	lop "github.com/samber/lo/parallel"
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
	NumTownBlocks uint32  `json:"numTownBlocks"`
	MaxTownBlocks uint32  `json:"maxTownBlocks"`
	BonusBlocks   uint16  `json:"bonusBlocks"`
	NumResidents  uint32  `json:"numResidents"`
	NumTrusted    uint16  `json:"numTrusted"`
	NumOutlaws    uint16  `json:"numOutlaws"`
	Balance       float32 `json:"balance"`
	ForSalePrice  float32 `json:"forSalePrice"`
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

// TODO: Implement this
type TownPerms struct {
	Build   [4]bool `json:"build"`
	Destroy [4]bool `json:"destroy"`
	Switch  [4]bool `json:"switch"`
	ItemUse [4]bool `json:"itemUse"`
	Flags   struct {
		PVP        bool `json:"pvp"`
		Explosions bool `json:"explosions"`
		Fire       bool `json:"fire"`
		Mobs       bool `json:"mobs"`
	} `json:"flags"`
}

type TownInfo struct {
	Entity
	Board       string     `json:"board"`
	Founder     string     `json:"founder"`
	Wiki        string     `json:"wiki"`
	Mayor       Entity     `json:"mayor"`
	Nation      *Entity    `json:"nation"`
	Residents   []Entity   `json:"residents"`
	Timestamps  Timestamps `json:"timestamps"`
	Status      TownStatus `json:"status"`
	Stats       TownStats  `json:"stats"`
	Coordinates TownCoords `json:"coordinates"`
	Ranks       TownRanks  `json:"ranks"`
	Trusted     []Entity   `json:"trusted"`
	Outlaws     []Entity   `json:"outlaws"`
	Perms       TownPerms  `json:"perms"`
}

func (t TownInfo) Bal() float32 {
	return t.Stats.Balance
}

func (t TownInfo) Area() uint32 {
	return t.Stats.MaxTownBlocks
}

func (t TownInfo) OverclaimedString() string {
	return strconv.FormatBool(t.Status.Overclaimed)
}

func (t TownInfo) GetResidentNames(alphabetical bool) []string {
	names := lop.Map(t.Residents, func(t Entity, _ int) string {
		return t.Name
	})

	if alphabetical {
		slices.Sort(names)
	}

	return names
}
