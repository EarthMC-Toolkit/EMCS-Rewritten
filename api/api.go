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
	"strings"

	"github.com/samber/lo/parallel"
)

func QueryOnlinePlayers() ([]oapi.PlayerInfo, error) {
	online, err := oapi.QueryList(oapi.ENDPOINT_ONLINE).Execute()
	if err != nil {
		return nil, err
	}

	ids := parallel.Map(online, func(p oapi.Entity, _ int) string {
		return p.UUID
	})

	players, errs, _ := oapi.QueryPlayers(ids...).ExecuteConcurrent()
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(players) == 0 {
		return nil, errors.New("failed to query online players, received 0 players from QueryConcurrent")
	}

	return players, nil
}

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

	ids := parallel.Map(visible, func(p mapi.MapPlayer, _ int) string {
		return p.UUID
	})

	players, errs, _ := oapi.QueryPlayers(ids...).ExecuteConcurrent()
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(players) == 0 {
		return nil, errors.New("failed to query visible players, received 0 players from QueryConcurrent")
	}

	return players, nil
}

// Runs a GET query for the town list, then POST queries every town concurrently using its UUID.
//
// Total number of requests sent should be 1+(total towns/QUERY_LIMIT).
func QueryAllTowns() ([]oapi.TownInfo, error) {
	tlist, err := oapi.QueryList(oapi.ENDPOINT_TOWNS).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to query all towns, could not get initial list\n%v", err)
	}

	ids := parallel.Map(tlist, func(e oapi.Entity, _ int) string {
		return e.UUID
	})

	towns, errs, _ := oapi.QueryTowns(ids...).ExecuteConcurrent()
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(towns) == 0 {
		return nil, errors.New("failed to query all towns, received 0 towns from QueryConcurrent")
	}

	return towns, nil
}

func QueryTown(townName string) (*oapi.TownInfo, error) {
	towns, err := oapi.QueryTowns(strings.ToLower(townName)).Execute()
	if err != nil {
		return nil, err
	}
	if len(towns) == 0 {
		return nil, nil
	}

	return &towns[0], nil
}

func QueryNation(nationName string) (*oapi.NationInfo, error) {
	nations, err := oapi.QueryNations(strings.ToLower(nationName)).Execute()
	if err != nil {
		return nil, err
	}
	if len(nations) == 0 {
		return nil, nil
	}

	return &nations[0], nil
}

func QueryPlayer(playerName string) (*oapi.PlayerInfo, error) {
	players, err := oapi.QueryPlayers(strings.ToLower(playerName)).Execute()
	if err != nil {
		return nil, err
	}
	if len(players) == 0 {
		return nil, nil
	}

	return &players[0], nil
}
