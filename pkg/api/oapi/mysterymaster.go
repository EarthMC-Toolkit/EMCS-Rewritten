package oapi

type ChangeDirection string

const (
	ChangeUp   ChangeDirection = "UP"
	ChangeDown ChangeDirection = "DOWN"
)

type MysteryMaster struct {
	Entity
	Change *ChangeDirection `json:"change"`
}

func (mm MysteryMaster) GetUUID() string {
	return mm.UUID
}
