package database

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

type AllianceColours struct {
	Fill    *string `json:"fill"`
	Outline *string `json:"outline"`
}

type AllianceOptionals struct {
	Leaders    *[]string        `json:"leaders,omitempty"`
	ImageURL   *string          `json:"imageURL,omitempty"`
	DiscordURL *string          `json:"discordURL,omitempty"`
	Colours    *AllianceColours `json:"colours,omitempty"`
}

type Alliance struct {
	ID               uint64            `json:"id"` // First 48bits = ms timestamp. Extra 16bits = randomness.
	Identifier       string            `json:"identifier"`
	Label            string            `json:"label"`
	RepresentativeID *uint64           `json:"representativeID"`
	Children         []string          `json:"children"`
	OwnNations       []string          `json:"ownNations"`
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"`
	Optional         AllianceOptionals `json:"optional"`
}

func (a *Alliance) CreatedTimestamp() uint64 {
	return a.ID >> 16
}

// ================================== DATABASE INTERACTION ==================================
const ALLIANCES_KEY_PREFIX = "alliances/"

// Finds the alliance by its key. ident is automatically lowercased for case-insensitive lookup.
// The actual `Identifier` property on the alliance will still have its original casing.
func GetAllianceByIdentifier(mapDB *badger.DB, ident string) (*Alliance, error) {
	return GetInsensitive[Alliance](mapDB, ALLIANCES_KEY_PREFIX+ident)
}

func PutAlliance(mapDB *badger.DB, a *Alliance) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("error putting alliance '%s' into db at %s:\n%v", a.Identifier, mapDB.Opts().Dir, err)
	}

	return PutInsensitive(mapDB, ALLIANCES_KEY_PREFIX+a.Identifier, data)
}

func DumpAlliances(mapDB *badger.DB) error {
	return mapDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			key := string(item.Key())
			if !strings.HasPrefix(key, ALLIANCES_KEY_PREFIX) {
				continue // Not an alliance, no need to dump.
			}

			var data Alliance
			val, _ := item.ValueCopy(nil)
			if err := json.Unmarshal(val, &data); err != nil {
				fmt.Printf("\n%s\n%s\n\n", key, val)
				continue
			}

			valPretty, _ := json.MarshalIndent(data, "", "  ")
			fmt.Printf("\n%s\n%s\n\n", key, valPretty)
		}

		return nil
	})
}
