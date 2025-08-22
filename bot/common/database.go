package common

type Alliance struct {
	UUID           string   `json:"uuid"`
	Label          string   `json:"label"`
	Keywords       []string `json:"keywords"`
	Representative string   `json:"representative"`
	Leaders        []string `json:"leaders,omitempty"`
	ImageURL       string   `json:"imageURL,omitempty"`
	Children       []string `json:"children,omitempty"`
	Nations        []string `json:"ownNations"`
	Discord        string   `json:"discord,omitempty"`
	Colours        *Colours `json:"colours,omitempty"`
	LastUpdated    uint64   `json:"lastUpdated"`
}

type Colours struct {
	Fill    string `json:"fill"`
	Outline string `json:"outline"`
}
