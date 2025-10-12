package oapi

import (
	"encoding/json"
	"fmt"
)

type QuarterType string

const (
	QuarterTypeApartment QuarterType = "APARTMENT"
	QuarterTypeInn       QuarterType = "INN"
	QuarterTypeStation   QuarterType = "STATION"
)

func (qt *QuarterType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case
		string(QuarterTypeApartment),
		string(QuarterTypeInn),
		string(QuarterTypeStation):

		*qt = QuarterType(s)

		return nil
	default:
		return fmt.Errorf("invalid quarter type: %s", s)
	}
}

func (qt QuarterType) Valid() bool {
	switch qt {
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
	Price        *float32 `json:"price"`
	Volume       uint64   `json:"volume"`
	NumCuboids   uint64   `json:"numCuboids"`
	ParticleSize *float32 `json:"particleSize"`
}

type QuarterCuboid struct {
	Pos1 []int32 `json:"pos1"`
	Pos2 []int32 `json:"pos2"`
}

type Quarter struct {
	Entity
	Type       QuarterType           `json:"type"`
	Creator    *string               `json:"creator"` // Just a UUID. Should be entity but whatever, OAPI is weird.
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
