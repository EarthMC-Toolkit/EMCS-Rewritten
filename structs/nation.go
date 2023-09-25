package structs

type NationStatus struct {
	Public			bool 	`json:"isPublic"`
	Open			bool 	`json:"isOpen"`
	Neutral			bool 	`json:"isNeutral"`
}

type NationStats struct {
	NumTownBlocks	int 	`json:"numTownBlocks"`
	NumResidents	int 	`json:"numResidents"`
	NumTowns		int 	`json:"numTowns"`
	Balance			float32 `json:"balance"`
}

type NationRanks struct {
	Chancellor		[]string 	`json:"Chancellor,omitempty"`
	Diplomat		[]string 	`json:"Diplomat,omitempty"`
	Colonist		[]string 	`json:"Colonist,omitempty"`
}

type NationInfo struct {
	Name 			*string			`json:"name"`
	UUID 			string			`json:"uuid"`
	King 			string			`json:"king"`
	Board 			string			`json:"board"`
	HexColor 		string			`json:"mapColorHexCode"`
	Capital 		string			`json:"capital"`
	Residents		[]string 		`json:"residents"`
	Towns			[]string 		`json:"towns"`
	Allies			[]string 		`json:"allies"`
	Ranks			NationRanks 	`json:"ranks"`
	Timestamps		Timestamps 		`json:"timestamps"`
	Status			NationStatus	`json:"status"`
	Stats			NationStats		`json:"stats"`
	Spawn			Spawn			`json:"spawn"`
}