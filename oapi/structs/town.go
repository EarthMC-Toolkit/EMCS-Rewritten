package structs

import "emcsrw/utils"

type TownStatus struct {
	Public      bool `json:"isPublic"`
	Open        bool `json:"isOpen"`
	Neutral     bool `json:"isNeutral"`
	Capital     bool `json:"isCapital"`
	Overclaimed bool `json:"isOverClaimed"`
	Ruined      bool `json:"isRuined"`
}

type TownStats struct {
	NumTownBlocks int16   `json:"numTownBlocks"`
	MaxTownBlocks int16   `json:"maxTownBlocks"`
	NumResidents  int16   `json:"numResidents"`
	Balance       float32 `json:"balance"`
}

type TownCoords struct {
	Home  []int `json:"home"`
	Spawn Spawn `json:"spawn"`
}

type TownRanks struct {
	Councillor []string `json:"Councillor,omitempty"`
	Builder    []string `json:"Builder,omitempty"`
	Recruiter  []string `json:"Recruiter,omitempty"`
	Police     []string `json:"Police,omitempty"`
	TaxExempt  []string `json:"Tax-exempt,omitempty"`
	Treasurer  []string `json:"Treasurer,omitempty"`
}

// TODO: Implement this
type TownPerms struct {
	
}

type TownInfo struct {
	Name		*string    	`json:"name"`
	UUID        string     	`json:"uuid"`
	Mayor       string     	`json:"mayor"`
	Board       string     	`json:"board"`
	Founder     string     	`json:"founder"`
	HexColor    string     	`json:"mapColorHexCode"`
	Nation      string     	`json:"nation,omitempty"`
	Residents   []string   	`json:"residents"`
	Timestamps  Timestamps 	`json:"timestamps"`
	Status      TownStatus 	`json:"status"`
	Stats       TownStats  	`json:"stats"`
	Coordinates TownCoords 	`json:"coordinates"`
	Ranks       TownRanks  	`json:"ranks"`
	Trusted     []string   	`json:"trusted"`
	Perms       any        	`json:"perms"`
}

func (t TownInfo) Bal() float32 {
	return t.Stats.Balance
}

func (t TownInfo) Area() int16 {
	return t.Stats.MaxTownBlocks
}

func (t TownInfo) ParseHexColor() int {
	return utils.HexToInt(t.HexColor)
}