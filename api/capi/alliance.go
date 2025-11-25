package capi

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"

	"github.com/samber/lo"
)

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders     []string         `json:"leaders"` // All UUIDs of alliance leaders that exist on EMC.
	ImageURL    *string          `json:"imageURL"`
	DiscordCode *string          `json:"discordCode"`
	Colours     *AllianceColours `json:"colours"`
}

type Alliance struct {
	UUID             uint64            `json:"uuid"`             // First 48bits = ms timestamp. Extra 16bits = randomness.
	Identifier       string            `json:"identifier"`       // Case-insensitive colloquial short name for lookup.
	Label            string            `json:"label"`            // Full name for display purposes.
	RepresentativeID *string           `json:"representativeID"` // Discord ID of the user representing this alliance.
	Parent           *string           `json:"parentAlliance"`   // The Identifier (not UUID) of the parent alliance this alliance is a child of.
	OwnNations       []string          `json:"ownNations"`       // UUIDs of nations in THIS alliance only.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"` // Unix timestamp (ms) at which the last update was made to this alliance.
	Optional         AllianceOptionals `json:"optional"`         // Extra properties that are not required for basic alliance functionality.
	Type             string            `json:"type"`             // Type of alliance. See AllianceType consts.
}

func ParseAlliance(a database.Alliance, nationStore *store.Store[oapi.NationInfo], entitiesStore *store.Store[oapi.EntityList]) Alliance {
	leaderNames := a.GetLeaderNames(entitiesStore)
	nationNames := lo.Map(a.GetOwnNations(nationStore), func(n oapi.NationInfo, _ int) string {
		return n.Name
	})

	return Alliance{
		UUID:             a.UUID,
		Identifier:       a.Identifier,
		Label:            a.Label,
		RepresentativeID: a.RepresentativeID,
		Parent:           a.Parent,
		OwnNations:       nationNames,
		UpdatedTimestamp: a.UpdatedTimestamp,
		Optional: AllianceOptionals{
			Leaders:     leaderNames,
			ImageURL:    a.Optional.ImageURL,
			DiscordCode: a.Optional.DiscordCode,
			Colours:     (*AllianceColours)(a.Optional.Colours),
		},
		Type: string(a.Type),
	}
}
