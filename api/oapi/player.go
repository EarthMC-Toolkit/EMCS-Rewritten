package oapi

type PlayerTimestamps struct {
	Timestamps
	JoinedTownAt *uint64 `json:"joinedTownAt"`
	LastOnline   *uint64 `json:"lastOnline"`
}

type PlayerStatus struct {
	IsOnline  bool `json:"isOnline"`
	IsNPC     bool `json:"isNPC"`
	IsMayor   bool `json:"isMayor"`
	IsKing    bool `json:"isKing"`
	HasTown   bool `json:"hasTown"`
	HasNation bool `json:"hasNation"`
}

type PlayerStats struct {
	Balance    float32 `json:"balance"`
	NumFriends uint16  `json:"numFriends"`
}

type ResidentRanks struct {
	Town   []string `json:"townRanks"`
	Nation []string `json:"nationRanks"`
}

type PlayerInfo struct {
	Name       string               `json:"name"`
	UUID       string               `json:"uuid"`
	Title      string               `json:"title,omitempty"`
	Surname    string               `json:"surname,omitempty"`
	Town       EntityNullableValues `json:"town"`
	Nation     EntityNullableValues `json:"nation"`
	Timestamps PlayerTimestamps     `json:"timestamps"`
	Status     PlayerStatus         `json:"status"`
	Stats      PlayerStats          `json:"stats"`
	Ranks      ResidentRanks        `json:"ranks"`
	Friends    []Entity             `json:"friends,omitempty"`
	Perms      Perms                `json:"perms"`
}
