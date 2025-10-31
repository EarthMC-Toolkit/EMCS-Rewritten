package store

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
	UUID             uint64            `json:"uuid"`             // First 48bits = ms timestamp. Extra 16bits = randomness.
	Identifier       string            `json:"identifier"`       // Case-insensitive colloquial short name for lookup.
	Label            string            `json:"label"`            // Full name for display purposes.
	RepresentativeID *string           `json:"representativeID"` // Discord ID of the user representing this alliance.
	Children         []string          `json:"children"`         // UUIDs of child alliances.
	OwnNations       []string          `json:"ownNations"`       // UUIDs of nations in this alliance, EXCLUDING ones in child alliances.
	UpdatedTimestamp *uint64           `json:"updatedTimestamp"` // Unix timestamp (ms) at which the last update was made to this alliance.
	// Type          string      	   `json:"type"`          	 // Type of alliance (sub, mega, pact). 
	Optional         AllianceOptionals `json:"optional"`         // Extra properties that are not required for basic alliance functionality.
}

func (a Alliance) CreatedTimestamp() uint64 {
	return a.UUID >> 16
}

// ================================== DATABASE INTERACTION ==================================
//const ALLIANCES_KEY_PREFIX = "alliances/"

// Finds the alliance by its key. ident is automatically lowercased for case-insensitive lookup.
// The actual `Identifier` property on the alliance will still have its original casing.
// func GetAllianceByIdentifier(allianceStore *Store[Alliance], ident string) (*Alliance, error) {
// 	a, err := allianceStore.GetKey(ident)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return a, nil
// }

// func DumpAlliances(mapDB *badger.DB) error {
// 	return mapDB.View(func(txn *badger.Txn) error {
// 		it := txn.NewIterator(badger.DefaultIteratorOptions)
// 		defer it.Close()
// 		for it.Rewind(); it.Valid(); it.Next() {
// 			item := it.Item()

// 			key := string(item.Key())
// 			if !strings.HasPrefix(key, ALLIANCES_KEY_PREFIX) {
// 				continue // Not an alliance, no need to dump.
// 			}

// 			var data Alliance
// 			val, _ := item.ValueCopy(nil)
// 			if err := json.Unmarshal(val, &data); err != nil {
// 				fmt.Printf("\n%s\n%s\n\n", key, val)
// 				continue
// 			}

// 			valPretty, _ := json.MarshalIndent(data, "", "  ")
// 			fmt.Printf("\n%s\n%s\n\n", key, valPretty)
// 		}

// 		return nil
// 	})
// }
