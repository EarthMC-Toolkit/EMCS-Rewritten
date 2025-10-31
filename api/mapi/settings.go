package mapi

import "emcsrw/utils/requests"

const SETTINGS_URL = "https://map.earthmc.net/tiles/minecraft_overworld/settings.json"

type SettingsResponse struct {
}

func GetSettings() (any, error) {
	return requests.JsonGetRequest[SettingsResponse](SETTINGS_URL)
}
