package structs

type Timestamps struct {
	Registered		float64 `json:"registered"`
	JoinedNationAt	float64	`json:"joinedNationAt,omitempty"`
}