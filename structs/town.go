package structs

type TownStatus struct {
	Public			bool 	`json:"isPublic"`
	Open			bool 	`json:"isOpen"`
	Neutral			bool 	`json:"isNeutral"`
	Capital			bool	`json:"isCapital"`
	Overclaimed		bool	`json:"isOverClaimed"`
	Ruined			bool 	`json:"isRuined"`
}

type TownStats struct {
	NumTownBlocks	int 	`json:"numTownBlocks"`
	MaxTownBlocks	int 	`json:"maxTownBlocks"`
	NumResidents	int 	`json:"numResidents"`
	Balance			float32 `json:"balance"`
}

type TownPerms struct {

}

type TownCoords struct {
	Home 			[]string	`json:"home"`
	Spawn struct {
		World		string		`json:"world"`
		X			float32		`json:"x"`
		Y			float32		`json:"y"`
		Z			float32		`json:"z"`
		Pitch		float32		`json:"pitch"`
		Yaw			float32		`json:"yaw"`
	} `json:"spawn"`
}

type TownInfo struct {
	Name 			string			`json:"name"`
	UUID 			string			`json:"uuid"`
	Mayor 			string			`json:"mayor"`
	Board 			string			`json:"board"`
	Founder 		string			`json:"founder"`
	HexColor 		string			`json:"mapColorHexCode"`
	Nation 			string			`json:"nation"`
	Residents		[]string 		`json:"residents"`
	Timestamps		Timestamps 		`json:"timestamps"`
	Status			TownStatus		`json:"status"`
	Stats			TownStats		`json:"stats"`
	Coordinates		TownCoords		`json:"coordinates"`
	//Perms			TownPerms		`json:"perms"`
}