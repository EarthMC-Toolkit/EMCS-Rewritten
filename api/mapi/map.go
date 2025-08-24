package mapi

import "emcsrw/utils"

const MARKERS_URL = "https://map.earthmc.net/tiles/minecraft_overworld/markers.json"

type MarkersResponse struct {
}

func GetMarkers() (any, error) {
	res, err := utils.JsonGetRequest[MarkersResponse](MARKERS_URL)

	return res, err
}
