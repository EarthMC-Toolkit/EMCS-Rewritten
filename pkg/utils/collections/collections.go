package collections

import "maps"

// Uses the built-in copy function and outputs a shallow copy of the input slice.
//
// Elements are copied into a new slice, but if T is a reference type (e.g. pointer, map, slice),
// the references themselves are copied, not the underlying data.
func CopySlice[T any](value []T) []T {
	cpy := make([]T, len(value))
	copy(cpy, value)
	return cpy
}

// Returns a shallow copy of the input map while preserving its type.
// For example, if a StringSet is passed (underlying map), a StringSet will also be returned.
func CopyMap[K comparable, V any, M ~map[K]V](m M) M {
	cpy := make(M, len(m))
	maps.Copy(cpy, m)
	return cpy
}

// Compares two maps for equality based on their keys only.
func MapKeysEqual[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}

	return true
}

// Returns items in s1 but not in s2 based on keyFunc.
func DifferenceBy[T any, K comparable](s1 []T, s2 []T, keyFn func(T) K) ([]T, map[K]struct{}) {
	seen := make(map[K]struct{}, len(s2))
	for _, v := range s2 {
		seen[keyFn(v)] = struct{}{}
	}

	result := make([]T, 0)
	for _, v := range s1 {
		if _, ok := seen[keyFn(v)]; !ok {
			result = append(result, v)
		}
	}

	return result, seen
}
