package shared

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"strings"
)

// Describes a player who has less info than a regular player than
// one returned from the Official API. It is possible they have opted out.
type BasicPlayer struct {
	oapi.Entity
	Town *oapi.TownInfo
	// DisplayName *string
	// Location    *mapi.Location
}

// Attempt to get a player's basic info without using the Official API.
//
// This is useful when a player has chosen to opt-out and can no longer be queried.
func GetBasicPlayer(nameInput string, reslist, townlesslist oapi.EntityList) *BasicPlayer {
	var e *oapi.Entity
	var t *oapi.TownInfo

	isResident := false
	for uuid, name := range reslist {
		if strings.EqualFold(name, nameInput) {
			isResident = true
			e = &oapi.Entity{UUID: uuid, Name: name}
		}
	}

	if !isResident {
		// see if they are townless instead
		for uuid, name := range townlesslist {
			if strings.EqualFold(name, nameInput) {
				e = &oapi.Entity{UUID: uuid, Name: name}
			}
		}
	} else {
		townStore, _ := database.GetStoreForMap(ACTIVE_MAP, database.TOWNS_STORE)
		t, _ = findPlayerTown(e.Name, townStore)
	}

	// since townlesslist is from the players endpoint (which is able to be opted-out of),
	// check visible players as extra fallback in case they opted-out and are townless.
	// visiblePlayers, _ := mapi.GetVisiblePlayers()
	// player, visible := lo.Find(visiblePlayers, func(v mapi.MapPlayer) bool {
	// 	return name == v.Name
	// })

	// Doesn't exist at all
	if e == nil {
		return nil
	}

	return &BasicPlayer{
		Entity: *e,
		Town:   t,
	}
}

func findPlayerTown(name string, townStore *store.Store[oapi.TownInfo]) (*oapi.TownInfo, error) {
	return townStore.FindFirst(func(t oapi.TownInfo) bool {
		for _, res := range t.Residents {
			if res.Name == name {
				return true
			}
		}

		return false
	})
}
