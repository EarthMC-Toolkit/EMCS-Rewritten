package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
)

// Describes a player who has less info than a regular player than
// one returned from the Official API. It is possible they have opted out.
type BasicPlayer struct {
	oapi.Entity
	Town   *oapi.Entity
	Nation *oapi.Entity
}

func FindPlayerTown(name string, townStore *store.Store[oapi.TownInfo]) (*oapi.TownInfo, error) {
	return townStore.FindFirst(func(t oapi.TownInfo) bool {
		for _, res := range t.Residents {
			if res.Name == name {
				return true
			}
		}

		return false
	})
}
