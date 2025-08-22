package mapi

import "emcsrw/utils"

const PLAYERS_URL = "https://map.earthmc.net/tiles/players.json"

type Location struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

type OnlinePlayer struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	World       string `json:"world"`
	Yaw         int32  `json:"yaw"`
	Location
}

type PlayersResponse struct {
	Max     uint16         `json:"max"`
	Players []OnlinePlayer `json:"players"`
}

func GetOnlinePlayers() ([]OnlinePlayer, error) {
	res, err := utils.JsonGetRequest[PlayersResponse](PLAYERS_URL)
	if err != nil {
		return nil, err
	}

	return res.Players, nil
}
