package sets

import (
	"encoding/json"
)

// type SyncedStringSet struct {
// 	StringSet
// 	mu sync.Mutex
// }

type StringSet = GenericSet[string]
type GenericSet[K comparable] map[K]struct{}

func FromSlice[K comparable](values []K) GenericSet[K] {
	s := make(GenericSet[K], len(values))
	for _, v := range values {
		s[v] = struct{}{}
	}

	return s
}

// Return all elements in this set as a slice.
func (s GenericSet[K]) Keys() (keys []K) {
	for k := range s {
		keys = append(keys, k)
	}

	return
}

func (set GenericSet[K]) Append(v K) {
	set[v] = struct{}{}
}

func (set GenericSet[K]) AppendFunc(v K, f func(v K) K) {
	set[f(v)] = struct{}{}
}

// func (set GenericSet) AppendIfUnseen(dst *[]string, v string) {
// 	if _, ok := set[v]; !ok {
// 		set[v] = struct{}{}
// 		*dst = append(*dst, v)
// 	}
// }

// // Adds values to dst if they are not already present in this set.
// //
// // Similar to AppendIfUnseen() for a single value, this func will append all values from the input slice instead.
// // This is useful when an intermediate set is required to deduplicate X slice while adding elements from Y slice.
// func (set GenericSet) AppendSlice(dst *[]string, values []string) {
// 	for _, v := range values {
// 		set.AppendIfUnseen(dst, v)
// 	}
// }

// Slice marshals this set as a JSON array of strings.
func (s GenericSet[K]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Keys())
}

// UnmarshalJSON unmarshals a JSON array of strings into this set.
func (s *GenericSet[K]) UnmarshalJSON(data []byte) error {
	var keys []K
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	if *s == nil {
		*s = make(GenericSet[K], len(keys))
	}

	for _, k := range keys {
		(*s)[k] = struct{}{}
	}

	return nil
}
