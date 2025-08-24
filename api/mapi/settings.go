package mapi

import "emcsrw/utils"

const SETTINGS_URL = "https://map.earthmc.net/tiles/minecraft_overworld/settings.json"

type SettingsResponse struct {
}

func GetSettings() (any, error) {
	res, err := utils.JsonGetRequest[SettingsResponse](SETTINGS_URL)
	return res, err
}
