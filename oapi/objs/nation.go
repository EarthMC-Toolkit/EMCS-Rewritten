package objs

import "emcsrw/utils"

type NationStatus struct {
	Public  bool `json:"isPublic"`
	Open    bool `json:"isOpen"`
	Neutral bool `json:"isNeutral"`
}

type NationStats struct {
	NumTownBlocks int     `json:"numTownBlocks"`
	NumResidents  int     `json:"numResidents"`
	NumTowns      int     `json:"numTowns"`
	NumAllies     int     `json:"numAllies"`
	NumEnemies    int     `json:"numEnemies"`
	Balance       float32 `json:"balance"`
}

type NationRanks struct {
	Chancellor []Entity `json:"Chancellor,omitempty"`
	Colonist   []Entity `json:"Colonist,omitempty"`
	Diplomat   []Entity `json:"Diplomat,omitempty"`
}

type NationInfo struct {
	Entity
	King             Entity       `json:"king"`
	Board            string       `json:"board"`
	Wiki             *string      `json:"wiki"`
	MapColourFill    string       `json:"dynmapColour"`
	MapColourOutline string       `json:"dynmapOutline"`
	Capital          Entity       `json:"capital"`
	Residents        []Entity     `json:"residents"`
	Towns            []Entity     `json:"towns"`
	Allies           []Entity     `json:"allies"`
	Enemies          []Entity     `json:"enemies"`
	Sanctioned       []Entity     `json:"sanctioned"`
	Ranks            NationRanks  `json:"ranks"`
	Timestamps       Timestamps   `json:"timestamps"`
	Status           NationStatus `json:"status"`
	Stats            NationStats  `json:"stats"`
	Spawn            Spawn        `json:"spawn"`
}

func (n NationInfo) Bal() float32 {
	return n.Stats.Balance
}

// Returns the fill colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) FillColourInt() int {
	return utils.HexToInt(n.MapColourFill)
}

// Returns the outline colour of the nation on the map as an int instead of a HEX string.
func (n NationInfo) OutlineColourInt() int {
	return utils.HexToInt(n.MapColourOutline)
}
