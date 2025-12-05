package mapi

import (
	"emcsrw/utils/requests"
)

const MAP_DOMAIN = "https://map.earthmc.net"
const MARKERS_URL = MAP_DOMAIN + "/tiles/minecraft_overworld/markers.json"

type MarkersResponse struct {
}

type ParsedMarker struct {
}

func GetMarkers() (map[string]ParsedMarker, error) {
	_, err := requests.JsonGetRequest[MarkersResponse](MARKERS_URL)

	// parse markers

	return nil, err
}
