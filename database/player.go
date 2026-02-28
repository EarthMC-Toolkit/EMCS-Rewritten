package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
)

type RankType string

const (
	RankTypeResident RankType = "Resident"
	RankTypeMayor    RankType = "Mayor"
	RankTypeLeader   RankType = "Nation Leader"
)

// Describes a player who has less info than a regular player (from the Official API).
// It is possible that this player has opted-out.
type BasicPlayer struct {
	oapi.Entity
	Town   *oapi.Entity `json:"town,omitempty"`
	Nation *oapi.Entity `json:"nation,omitempty"`
	Rank   *RankType    `json:"rank,omitempty"`
}

// Returns the player Rank (one of [RankType]) as its string value. Defaults to "Townless" if nil/no rank.
func (p *BasicPlayer) RankString() string {
	if p.Rank != nil {
		return string(*p.Rank)
	}
	return "Townless"
}

func NewBasicPlayerEntity(id, n string) BasicPlayer {
	return BasicPlayer{
		Entity: oapi.Entity{UUID: id, Name: n},
		Rank:   nil,
	}
}

func FindPlayerTown(name string, townStore *store.Store[oapi.TownInfo]) (*oapi.TownInfo, error) {
	return townStore.Find(func(t oapi.TownInfo) bool {
		for _, res := range t.Residents {
			if res.Name == name {
				return true
			}
		}

		return false
	})
}
