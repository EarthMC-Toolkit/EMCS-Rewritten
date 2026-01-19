package oapi

import "slices"

type PlayerTimestamps struct {
	Timestamps
	JoinedTownAt *uint64 `json:"joinedTownAt"`
	LastOnline   *uint64 `json:"lastOnline"`
}

type PlayerStatus struct {
	IsOnline  bool `json:"isOnline"`
	IsNPC     bool `json:"isNPC"`
	IsMayor   bool `json:"isMayor"`
	IsKing    bool `json:"isKing"`
	HasTown   bool `json:"hasTown"`
	HasNation bool `json:"hasNation"`
}

type PlayerStats struct {
	Balance    float32 `json:"balance"`
	NumFriends uint16  `json:"numFriends"`
}

type ResidentRanks struct {
	Town   []string `json:"townRanks"`
	Nation []string `json:"nationRanks"`
}

type PlayerInfo struct {
	Entity
	Title         *string              `json:"title,omitempty"`
	Surname       *string              `json:"surname,omitempty"`
	FormattedName string               `json:"formattedName,omitempty"`
	About         *string              `json:"about,omitempty"`
	Town          EntityNullableValues `json:"town"`
	Nation        EntityNullableValues `json:"nation"`
	Timestamps    PlayerTimestamps     `json:"timestamps"`
	Status        PlayerStatus         `json:"status"`
	Stats         PlayerStats          `json:"stats"`
	Ranks         ResidentRanks        `json:"ranks"`
	Friends       []Entity             `json:"friends,omitempty"`
	Perms         Perms                `json:"perms"`
}

func (p PlayerInfo) GetUUID() string {
	return p.UUID
}

func (p PlayerInfo) GetRank() string {
	if p.Status.IsKing {
		return "Nation Leader"
	}
	if p.Status.IsMayor {
		return "Mayor"
	}

	return "Resident"
}

// In order of priority so we show "better" roles first.
var TOWNY_ROLES = []string{"Councillor", "Assistant", "Helper", "Builder", "Recruiter"}

func (p PlayerInfo) GetRankOrRole() string {
	if p.Status.IsKing {
		return "Nation Leader"
	}
	if p.Status.IsMayor {
		return "Mayor"
	}

	for _, role := range TOWNY_ROLES {
		if slices.Contains(p.Ranks.Town, role) {
			return role
		}
	}

	return "Resident"
}
