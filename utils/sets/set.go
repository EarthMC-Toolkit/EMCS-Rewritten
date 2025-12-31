package sets

import (
	"encoding/json"
)

type Set[K comparable] map[K]struct{}

func FromSlice[K comparable](values []K) Set[K] {
	s := make(Set[K], len(values))
	for _, v := range values {
		s.Append(v)
	}

	return s
}

// Return all elements in this set as a slice.
func (s Set[K]) Keys() []K {
	keys := make([]K, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	return keys
}

func (s Set[K]) Append(key K) {
	s[key] = struct{}{}
}

func (s Set[K]) AppendFunc(key K, f func(key K) K) {
	s[f(key)] = struct{}{}
}

// Slice marshals this set as a JSON array of strings.
func (s Set[K]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Keys())
}

// UnmarshalJSON unmarshals a JSON array of strings into this set.
func (s *Set[K]) UnmarshalJSON(data []byte) error {
	var keys []K
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	if *s == nil {
		*s = make(Set[K], len(keys))
	}

	for _, k := range keys {
		(*s)[k] = struct{}{}
	}

	return nil
}
