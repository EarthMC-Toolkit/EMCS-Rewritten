package sets

import (
	"encoding/json"
	"maps"
)

// type SyncedStringSet struct {
// 	StringSet
// 	mu sync.Mutex
// }

type StringSet map[string]struct{}

// Return all elements in this set as a slice.
func (s StringSet) Keys() (keys []string) {
	for k := range s {
		keys = append(keys, k)
	}

	return
}

func (set StringSet) Append(v string) {
	set[v] = struct{}{}
}

// func (set StringSet) AppendIfUnseen(dst *[]string, v string) {
// 	if _, ok := set[v]; !ok {
// 		set[v] = struct{}{}
// 		*dst = append(*dst, v)
// 	}
// }

// // Adds values to dst if they are not already present in this set.
// //
// // Similar to AppendIfUnseen() for a single value, this func will append all values from the input slice instead.
// // This is useful when an intermediate set is required to deduplicate X slice while adding elements from Y slice.
// func (set StringSet) AppendSlice(dst *[]string, values []string) {
// 	for _, v := range values {
// 		set.AppendIfUnseen(dst, v)
// 	}
// }

// Slice marshals this set as a JSON array of strings.
func (s StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Keys())
}

// UnmarshalJSON unmarshals a JSON array of strings into this set.
func (s *StringSet) UnmarshalJSON(data []byte) error {
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	if *s == nil {
		*s = make(StringSet, len(keys))
	}

	for _, k := range keys {
		(*s)[k] = struct{}{}
	}

	return nil
}

// Constructs a new StringSet from this set.
func (s StringSet) Copy() StringSet {
	copy := make(StringSet, len(s))
	maps.Copy(copy, s)

	return copy
}
