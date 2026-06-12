package mapi

import (
	"emcsrw/pkg/utils/requests"
	"fmt"
)

const PLAYERS_URL = MAP_DOMAIN + "/tiles/players.json"

type Location struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

func (loc Location) ToMapLink(zoom uint8) string {
	return fmt.Sprintf("%s?x=%d&z=%d&zoom=%d", MAP_DOMAIN, loc.X, loc.Z, zoom)
}

type MapPlayer struct {
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	DisplayName *string `json:"display_name"`
	World       string  `json:"world"`
	Yaw         int32   `json:"yaw"`
	Location
}

type PlayersResponse struct {
	Max     uint16      `json:"max"`
	Players []MapPlayer `json:"players"`
}

// TODO: Maybe return map instead, using UUID as key for faster lookup?
func GetVisiblePlayers() ([]MapPlayer, error) {
	res, err := requests.JsonGet[PlayersResponse](PLAYERS_URL)
	if err != nil {
		return nil, err
	}

	return res.Players, nil
}

// Convert a non-hyphenated UUID to one containing hyphens
// which most things (including the EMC API) expect.
//
// If the input uuid is not 32 characters in length, it will be returned as is.
func NormalizeUUID(uuid string) string {
	if len(uuid) != 32 {
		return uuid
	}

	return uuid[:8] + "-" +
		uuid[8:12] + "-" +
		uuid[12:16] + "-" +
		uuid[16:20] + "-" +
		uuid[20:]
}
