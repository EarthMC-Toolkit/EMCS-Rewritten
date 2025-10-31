package mapi

import "emcsrw/utils/requests"

const PLAYERS_URL = "https://map.earthmc.net/tiles/players.json"

type Location struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

type MapPlayer struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	World       string `json:"world"`
	Yaw         int32  `json:"yaw"`
	Location
}

type PlayersResponse struct {
	Max     uint16      `json:"max"`
	Players []MapPlayer `json:"players"`
}

// TODO: Maybe return map instead. Use UUID as key for faster lookup?
func GetVisiblePlayers() ([]MapPlayer, error) {
	res, err := requests.JsonGetRequest[PlayersResponse](PLAYERS_URL)
	if err != nil {
		return nil, err
	}

	return res.Players, nil
}
