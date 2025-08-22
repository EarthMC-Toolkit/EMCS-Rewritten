package database

type AllianceColours struct {
	Fill    string `json:"fill"`
	Outline string `json:"outline"`
}

type Alliance struct {
	UUID           string           `json:"uuid"`
	Label          string           `json:"label"`
	Keywords       []string         `json:"keywords"`
	Representative string           `json:"representative"`
	Leaders        []string         `json:"leaders,omitempty"`
	ImageURL       string           `json:"imageURL,omitempty"`
	Children       []string         `json:"children,omitempty"`
	Nations        []string         `json:"ownNations"`
	Discord        string           `json:"discord,omitempty"`
	Colours        *AllianceColours `json:"colours,omitempty"`
	LastUpdated    uint64           `json:"lastUpdated"`
}

// TODO: Implement this.
func GetAlliances() map[string]Alliance {
	// Read alliances from `db` dir

	// Convert slice to map using uuid as key

	return nil
}
