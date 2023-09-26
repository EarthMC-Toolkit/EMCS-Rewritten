package structs

type ServerWorld struct {
	Storming		bool	`json:"hasStorm"`
	Thundering		bool	`json:"isThundering"`
	Time			int16	`json:"time"`
	FullTime		int32	`json:"fullTime"`
}

type ServerPlayers struct {
	Max				int16	`json:"maxPlayers"`
	OnlineTownless	int16	`json:"numOnlineTownless"`
	OnlinePlayers	int16	`json:"numOnlinePlayers"`
}

type ServerStats struct {
	Residents		int32	`json:"numResidents"`
	Townless		int32	`json:"numTownless"`
	Towns			int16	`json:"numTowns"`
	Nations			int16	`json:"numNations"`
	TownBlocks		int32	`json:"numTownBlocks"`
}

type ServerInfo struct {
	World		ServerWorld
	Players		ServerPlayers
	Stats		ServerStats
}