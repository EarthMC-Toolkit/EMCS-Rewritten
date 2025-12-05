package oapi

var rnao = [4]byte{'r', 'n', 'a', 'o'}

// Takes a single permission (slice of bools) and outputs a string in the RNAO (Resident, Nation, Ally, Outsider) format.
func encodePerm(perm [4]bool) string {
	b := make([]byte, 4)
	for i, v := range perm {
		if v {
			b[i] = rnao[i]
		} else {
			b[i] = '-'
		}
	}

	return string(b)
}

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

// Encodes all town permissions (Build, Destroy, Switch, ItemUse) into their RNAO string equivalents as seen in Towny.
//
// Here are some examples of what each one could look like:
//
//	"r-r-r-r"
//	"r-r----"
//	"--r-r--"
//	"-------"
func (p Perms) GetPermStrings() (string, string, string, string) {
	return encodePerm(p.Build), encodePerm(p.Destroy), encodePerm(p.Switch), encodePerm(p.ItemUse)
}

type EntityNullableValues struct {
	Name *string `json:"name"`
	UUID *string `json:"uuid"`
}

type Entity struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

// Maps an entity's UUID -> Name. Alternative to []oapi.Entity and usually preferred.
type EntityList = map[string]string
