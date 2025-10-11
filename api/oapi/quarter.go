package oapi

type QuarterType string

const (
	QuarterTypeApartment QuarterType = "APARTMENT"
	QuarterTypeInn       QuarterType = "INN"
	QuarterTypeStation   QuarterType = "STATION"
)

func (t QuarterType) Valid() bool {
	switch t {
	case QuarterTypeApartment, QuarterTypeInn, QuarterTypeStation:
		return true
	default:
		return false
	}
}

type QuarterTimestamps struct {
	Registered uint64  `json:"registered"`
	ClaimedAt  *uint64 `json:"claimedAt"`
}

type QuarterStatus struct {
	IsEmbassy bool `json:"isEmbassy"`
	IsForSale bool `json:"isForSale"`
}

type QuarterStats struct {
	Price        *int32  `json:"price"`
	Volume       uint64  `json:"volume"`
	NumCuboids   uint64  `json:"numCuboids"`
	ParticleSize *uint32 `json:"particleSize"`
}

type QuarterCuboid struct {
	Pos1 []int32 `json:"pos1"`
	Pos2 []int32 `json:"pos2"`
}

type Quarter struct {
	Entity
	Type       QuarterType           `json:"type"`
	Creator    EntityNullableValues  `json:"creator"`
	Owner      EntityNullableValues  `json:"owner"`
	Town       *EntityNullableValues `json:"town"`
	Nation     *EntityNullableValues `json:"nation"`
	Timestamps QuarterTimestamps     `json:"timestamps"`
	Status     QuarterStatus         `json:"status"`
	Stats      QuarterStats          `json:"stats"`
	Colour     []uint8               `json:"colour"` // [0]R [1]G [2]B [3]A
	Trusted    []Entity              `json:"trusted"`
	Cuboids    []QuarterCuboid
}

func (q Quarter) GetUUID() string {
	return q.UUID
}
