package common

import (
	"emcsrw/mapi"
	"emcsrw/oapi"

	lop "github.com/samber/lo/parallel"
)

// THIS FILE CONTAINS COMMON API UTILITIES USED THROUGHOUT THE BOT THAT
// MAY NOT FIT IN EITHER THE MAPI OR OAPI PACKAGES.
//
// IT SERVES AS A BRIDGE BETWEEN THE TWO WHERE NECESSARY.

// Uses the map api to get online players, chunks them into slices with max len of chunkSize, and queries official api for their full player info.
//
// Returns back the same list of online players as a single slice.
// Essentially, this acts as a conversion between []mapi.OnlinePlayer and []oapi.PlayerInfo.
func QueryOnlinePlayers(chunkSize uint8) ([]oapi.PlayerInfo, error) {
	ops, err := mapi.GetOnlinePlayers()
	if err != nil {
		return nil, err
	}

	opNames := lop.Map(ops, func(op mapi.OnlinePlayer, _ int) string {
		return op.Name
	})

	players, _ := oapi.QueryPlayersConcurrent(opNames, 100)

	return players, nil
}
