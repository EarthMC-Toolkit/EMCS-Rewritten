package utils

import "slices"

type KeySortOption[T any] struct {
	Compare func(a, b T) bool // returns true if a should come before b
}

// KeySort sorts slice s in-place by keys (order is kept).
// First key is the primary sort key, second key is less important, and so on.
func KeySort[T any](s []T, keys []KeySortOption[T]) []T {
	slices.SortFunc(s, func(a, b T) int {
		for _, k := range keys {
			if k.Compare(a, b) {
				return -1 // a comes before b
			}
			if k.Compare(b, a) {
				return 1 // b comes before a
			}
			// equal, continue to next key
		}

		return 0
	})

	return s
}

func SortToggledOn[T any](arr []T, rank func(T) bool) []T {
	slices.SortFunc(arr, func(a, b T) int {
		switch {
		case rank(a) == rank(b):
			return 0
		case rank(a):
			return -1
		default:
			return 1
		}
	})

	return arr
}

func RankSortAscending[T any](arr []T, rank func(T) int) []T {
	slices.SortFunc(arr, func(a, b T) int {
		return rank(a) - rank(b)
	})

	return arr
}

func RankSortDescending[T any](arr []T, rank func(T) int) []T {
	slices.SortFunc(arr, func(a, b T) int {
		return rank(b) - rank(a)
	})

	return arr
}
