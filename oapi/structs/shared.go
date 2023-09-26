package structs

type Timestamps struct {
	Registered		float64 `json:"registered"`
	JoinedNationAt	float64	`json:"joinedNationAt,omitempty"`
	LastOnline		float64	`json:"lastOnline,omitempty"`
}

type Spawn struct {
	World		string		`json:"world"`
	X			float32		`json:"x"`
	Y			float32		`json:"y"`
	Z			float32		`json:"z"`
	Pitch		float32		`json:"pitch"`
	Yaw			float32		`json:"yaw"`
}