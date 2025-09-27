// =========================================================================
// THIS PACKAGE CONTAINS COMMON API UTILITIES USED THROUGHOUT THE BOT
// THAT MAY NOT FIT IN EITHER THE MAPI OR OAPI PACKAGES.
//
// SOME FUNCS MAY COMMUNICATE WITH BOTH PACKAGES WHERE NEEDED, SO THIS
// PACKAGE WILL ACT AS A WIRE TO AVOID IMPORT CYCLES IN THOSE CASES.
// =========================================================================

package api

import (
	"emcsrw/api/mapi"
	"emcsrw/api/oapi"
	"errors"
	"fmt"

	lop "github.com/samber/lo/parallel"
)

// Uses the map API to get online players, chunks them into slices with max len of [oapi.PLAYERS_QUERY_LIMIT],
// and then queries the official API for their full player info.
//
// Returns back the same list of online players as a single slice.
// Essentially, this acts as a conversion between []mapi.OnlinePlayer and []oapi.PlayerInfo.
func QueryVisiblePlayers() ([]oapi.PlayerInfo, error) {
	visible, err := mapi.GetVisiblePlayers()
	if err != nil {
		return nil, err
	}

	names := lop.Map(visible, func(p mapi.MapPlayer, _ int) string {
		return p.Name
	})

	players, errs, _ := oapi.QueryConcurrent(names, oapi.QueryPlayers)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if len(players) == 0 {
		return nil, errors.New("failed to query all players, received 0 players from QueryConcurrent")
	}

	return players, nil
}

// Runs a GET query for the town list, then POST queries every town concurrently using its UUID.
//
// Total number of requests sent should be 1+(total towns/QUERY_LIMIT).
func QueryAllTowns() ([]oapi.TownInfo, error) {
	tlist, err := oapi.QueryList(oapi.ENDPOINT_TOWNS)
	if err != nil {
		return nil, fmt.Errorf("failed to query all towns, could not get initial list\n%v", err)
	}

	identifiers := lop.Map(tlist, func(e oapi.Entity, _ int) string {
		return e.UUID
	})

	towns, errs, _ := oapi.QueryConcurrent(identifiers, oapi.QueryTowns)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if len(towns) == 0 {
		return nil, errors.New("failed to query all towns, received 0 towns from QueryConcurrent")
	}

	return towns, nil
}

// TODO: Might not even need this since we can just do QueryConcurrent for nations
// using gathered data we got from doing QueryConcurrent for towns previously.
// func QueryAllNations() ([]oapi.NationInfo, error) {
// 	nlist, err := oapi.QueryList(oapi.ENDPOINT_NATIONS)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query all nations, could not get initial list\n%v", err)
// 	}

// 	identifiers := lop.Map(nlist, func(e oapi.Entity, _ int) string {
// 		return e.UUID
// 	})

// 	nations, _, _ := oapi.QueryConcurrent(identifiers, oapi.QueryNations)
// 	return nations, nil
// }
