package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
)

// Describes a player who has less info than a regular player (from the Official API).
// It is possible that this player has opted-out.
type BasicPlayer struct {
	oapi.Entity
	Town   *oapi.Entity `json:"town,omitempty"`
	Nation *oapi.Entity `json:"nation,omitempty"`
}

func NewBasicPlayerEntity(id, n string) BasicPlayer {
	return BasicPlayer{Entity: oapi.Entity{UUID: id, Name: n}}
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
