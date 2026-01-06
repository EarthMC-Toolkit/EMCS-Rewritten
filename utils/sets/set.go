package sets

import (
	"encoding/json"
	"maps"
)

type Set[K comparable] map[K]struct{}

func FromSlice[K comparable](keys []K) Set[K] {
	s := make(Set[K], len(keys))
	for _, k := range keys {
		s.Append(k)
	}

	return s
}

func (s Set[K]) Has(key K) bool {
	_, ok := s[key]
	return ok
}

// Adds key to this set.
// If the key needs to be modified or different based on condition, use AppendFunc instead.
func (s Set[K]) Append(key K) {
	s[key] = struct{}{}
}

// Passes the key to func f before adding it to this set.
func (s Set[K]) AppendFunc(key K, f func(key K) K) {
	s[f(key)] = struct{}{}
}

// Returns all elements in this set as a slice.
func (s Set[K]) Keys() []K {
	keys := make([]K, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	return keys
}

// Returns a new set containing all elements from s and the given sets.
func (s Set[K]) Union(sets ...Set[K]) Set[K] {
	merged := maps.Clone(s)
	for _, set := range sets {
		for k := range set {
			merged.Append(k)
		}
	}

	return merged
}

// Returns all elements in s that are not in other.
func (s Set[K]) Difference(other Set[K]) Set[K] {
	set := make(Set[K])
	for k := range s {
		if _, ok := other[k]; !ok {
			set.Append(k)
		}
	}

	return set
}

// Serializes this set's keys to a JSON array of strings.
func (s Set[K]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Keys())
}

// Deserializes a JSON array of strings, rebuilding this set.
func (s *Set[K]) UnmarshalJSON(data []byte) error {
	var keys []K
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	// Usually I'd just use "*s = FromSlice(keys)" but this allows us
	// to re-use the same slice to avoid an extra allocation ;)
	if *s != nil {
		for k := range *s {
			delete(*s, k) // clear all existing elements
		}
	} else {
		*s = make(Set[K], len(keys)) // we don't have one allocated, make one.
	}

	for _, k := range keys {
		(*s).Append(k)
	}

	return nil
}
