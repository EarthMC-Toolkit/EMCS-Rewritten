package mapi

import (
	"emcsrw/utils/requests"
)

const (
	MAP_DOMAIN  = "https://map.earthmc.net"
	MARKERS_URL = MAP_DOMAIN + "/tiles/minecraft_overworld/markers.json"
)

type LayerName string

const (
	LayerNameTerritory    LayerName = "Territory"
	LayerNameFoliaRegions LayerName = "Folia Regions"
	LayerNameChunkBorders LayerName = "Chunk Borders"
	LayerNameWorldBorder  LayerName = "World Border"
)

type Layer struct {
	Name      LayerName `json:"name"`
	ID        string    `json:"id"`
	Hide      bool      `json:"hide"`
	Control   bool      `json:"control"`
	Order     int       `json:"order"`
	ZIndex    int       `json:"z_index"`
	Timestamp int64     `json:"timestamp"`
	Markers   []Marker  `json:"markers"`
}

type Point2D struct {
	X int `json:"x"`
	Z int `json:"z"`
}

type Marker struct {
	Popup   *string   `json:"popup,omitempty"`
	Tooltip *string   `json:"tooltip,omitempty"`
	Type    string    `json:"type"`
	Color   any       `json:"color"` // false or HEX string including the # prefix
	Fill    any       `json:"fill"`  // false or HEX string including the # prefix
	Points  []Point2D `json:"points"`
}

type ParsedMarker struct {
}

func GetMarkers() (map[string]ParsedMarker, error) {
	_, err := requests.JsonGet[[]Layer](MARKERS_URL)

	// parse markers

	return nil, err
}
