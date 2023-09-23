package structs

type ResidentStatus struct {
	Online 	bool 			`json:"isOnline"`
}

// TODO: Fully implement this
type ResidentInfo struct {
	Name   	string   	 	`json:"name"`
	Status 	ResidentStatus 	`json:"status"`
}