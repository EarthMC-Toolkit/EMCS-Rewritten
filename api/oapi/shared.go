package oapi

type Timestamps struct {
	Registered uint64 `json:"registered"`
}

type Spawn struct {
	World string  `json:"world"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
	Z     float32 `json:"z"`
	Pitch float32 `json:"pitch"`
	Yaw   float32 `json:"yaw"`
}

type Perms struct {
	Build   [4]bool `json:"build"`
	Destroy [4]bool `json:"destroy"`
	Switch  [4]bool `json:"switch"`
	ItemUse [4]bool `json:"itemUse"`
	Flags   struct {
		PVP        bool `json:"pvp"`
		Explosions bool `json:"explosions"`
		Fire       bool `json:"fire"`
		Mobs       bool `json:"mobs"`
	} `json:"flags"`
}

type EntityNullableValues struct {
	Name *string `json:"name"`
	UUID *string `json:"uuid"`
}

type Entity struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}
