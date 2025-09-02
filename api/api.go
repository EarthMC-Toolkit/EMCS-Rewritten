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
	"fmt"

	lop "github.com/samber/lo/parallel"
)

// Uses the map API to get online players, chunks them into slices with max len of [oapi.PLAYERS_QUERY_LIMIT],
// and then queries the official API for their full player info.
//
// Returns back the same list of online players as a single slice.
// Essentially, this acts as a conversion between []mapi.OnlinePlayer and []oapi.PlayerInfo.
func QueryOnlinePlayers() ([]oapi.PlayerInfo, error) {
	ops, err := mapi.GetOnlinePlayers()
	if err != nil {
		return nil, err
	}

	opNames := lop.Map(ops, func(op mapi.OnlinePlayer, _ int) string {
		return op.Name
	})

	players, _, _ := oapi.QueryPlayersConcurrent(opNames, 0)
	return players, nil
}

func QueryAllTowns(save bool) ([]oapi.TownInfo, error) {
	_, err := oapi.QueryList(oapi.TOWNS_ENDPOINT)
	if err != nil {
		return nil, fmt.Errorf("failed to query all towns, could not get initial list\n%v", err)
	}

	// identifiers := lop.Map(tlist, func(e oapi.Entity, _ int) string {
	// 	return e.UUID
	// })

	// query towns concurrently
	

	// write to ~cwd/db/towns.json
	if save {

	}

	return nil, nil
}
