package mapi

import "emcsrw/utils/requests"

const SETTINGS_URL = MAP_DOMAIN + "/tiles/minecraft_overworld/settings.json"

type SettingsResponse struct {
	MarkerUpdateInterval uint16 `json:"marker_update_interval"`
	TilesUpdateInterval  uint16 `json:"tiles_update_interval"`
	PlayerTracker        struct {
		Enabled        bool   `json:"enabled"`
		Label          string `json:"label"`
		UpdateInterval uint16 `json:"update_interval"`
		Priority       int16  `json:"priority"`
		ZIndex         int    `json:"z_index"`
		ShowControls   bool   `json:"show_controls"`
		DefaultHidden  bool   `json:"default_hidden"`
		Nameplates     struct {
			Enabled    bool   `json:"enabled"`
			ShowHealth bool   `json:"show_health"`
			ShowArmour bool   `json:"show_armor"`
			ShowHeads  bool   `json:"show_heads"`
			HeadsURL   string `json:"heads_url"`
		}
	}
	Spawn struct {
		X int `json:"x"`
		Z int `json:"z"`
	}
	Zoom struct {
		Def   int8  `json:"def"`
		Max   uint8 `json:"max"`
		Extra uint8 `json:"extra"`
	}
}

func GetSettings() (any, error) {
	return requests.JsonGet[SettingsResponse](SETTINGS_URL)
}
