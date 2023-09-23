package structs

type ResidentStatus struct {
	Online 		bool 	`json:"isOnline"`
	NPC 		bool 	`json:"isNPC"`
}

type ResidentStats struct {
	Balance		float32 	`json:"balance"`
}

type ResidentRanks struct {
	Nation	NationRanks	`json:"nationRanks,omitempty"`
	Town	TownRanks	`json:"TownRanks,omitempty"`
}


// TODO: Fully implement this
type ResidentInfo struct {
	Name   		string   	 	`json:"name"`
	UUID   		string   	 	`json:"uuid"`
	Title   	string   	 	`json:"title,omitempty"`
	Surname   	string   	 	`json:"surname,omitempty"`
	Town   		string   	 	`json:"town,omitempty"`
	Nation   	string   	 	`json:"nation,omitempty"`
	Timestamps	Timestamps		`json:"timestamps"`
	Status 		ResidentStatus 	`json:"status"`
	Stats		ResidentStats	`json:"stats"`
	Ranks		ResidentRanks	`json:"ranks,omitempty"`
	Friends		[]string		`json:"friends,omitempty"`
	//Perms		ResidentPerms	`json:"perms"`
}