package utils

import (
	"cmp"
	"emcsrw/pkg/utils/sets"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/samber/lo"
)

// #region Slice deduplication
func DedupeSlice[T any, K comparable](keyFunc func(T) K, values []T) []T {
	d := NewSliceDeduper(keyFunc, values)
	return d.items
}

func NewSliceDeduper[T any, K comparable](keyFunc func(T) K, values []T) *SliceDeduper[T, K] {
	s := sets.Make[K](len(values))
	d := &SliceDeduper[T, K]{seen: s, keyFn: keyFunc}
	for _, v := range values {
		d.Append(v)
	}

	return d
}

// Deduper is a wrapper for a regular slice which only allows insertion
// into the slice based on a custom equality comparator.
type SliceDeduper[T any, K comparable] struct {
	items []T
	seen  sets.Set[K]
	keyFn func(T) K
}

// Appends v to the end of the slice if there is not already an element
// within the slice that satisfies the equality check
func (d *SliceDeduper[T, K]) Append(v T) bool {
	k := d.keyFn(v)
	if exists := d.seen.Has(k); exists {
		return false
	}

	d.seen.Add(k)
	d.items = append(d.items, v)
	return true
}

//#endregion

// Converts an amount of minutes into the closest matching unit (hr/min/sec) for display purposes. For example:
//
//	120  -> (2, 'hr')
//	5    -> (5, 'min')
//	0.5  -> (30, 'sec')
func HumanizeDuration(minutes float64) (float64, string) {
	if minutes >= 60 {
		return minutes / 60, "hr"
	}
	if minutes >= 1 {
		return minutes, "min"
	}

	return minutes * 60, "sec"
}

// Converts seconds into a human-readable duration string.
// Output formats:
//
//	"1hr, 5m and 10s"
//	"5m and 10s"
//	"10s"
func FormatElapsed(secs int64) string {
	hours := secs / 3600
	minutes := (secs % 3600) / 60
	seconds := secs % 60

	if hours > 0 {
		h := "hrs"
		if hours == 1 {
			h = "hr"
		}
		return fmt.Sprintf("`%d%s`, `%dm` and `%ds`", hours, h, minutes, seconds)
	}

	if minutes > 0 {
		return fmt.Sprintf("`%dm` and `%ds`", minutes, seconds)
	}

	return fmt.Sprintf("`%ds`", seconds)
}

// Formats a time.Time to a string in the format "Wed, Jan 2nd 3PM UTC".
func FormatTime(t time.Time) string {
	t = t.UTC() // so we can use the output as an anchor point for local timezones

	day := t.Day()
	suffix := "th"
	if day%10 == 1 && day != 11 {
		suffix = "st"
	} else if day%10 == 2 && day != 12 {
		suffix = "nd"
	} else if day%10 == 3 && day != 13 {
		suffix = "rd"
	}

	return fmt.Sprintf("%s, %s %d%s %dAM UTC",
		t.Weekday().String()[:3], // First three letters of the weekday word.
		t.Month().String()[:3],   // First three letters of the month word.
		day, suffix, t.Hour()%12,
	)
}

// Takes an amount of seconds and converts it to a string with any
// combination of hr/min/sec depending how long it takes.
// func FormatDuration(seconds int64) string {
// 	hours := seconds / 3600
// 	minutes := (seconds % 3600) / 60
// 	secs := seconds % 60

// 	if hours > 0 {
// 		return fmt.Sprintf("%dhrs, %dm and %ds", hours, minutes, secs)
// 	}

// 	if minutes > 0 {
// 		return fmt.Sprintf("%dm and %ds", minutes, secs)
// 	}

// 	return fmt.Sprintf("%ds", secs)
// }

// Check that `str` isn't gibberish and only has a combination of letters and numbers.
// If it is found to contain anything else, an empty string is returned.
func CheckAlphanumeric(str *string) string {
	if str == nil {
		return ""
	}

	return lo.Ternary(ContainsNonAlphanumeric(*str), "", *str)
}

func ContainsNonAlphanumeric(input string) bool {
	// Define a regular expression pattern to match non-alphanumeric characters
	pattern := regexp.MustCompile(`[^a-zA-Z0-9]`)

	// If there are matches, it means non-alphanumeric characters were found
	return pattern.MatchString(input)
}

func HexToInt(hex string) int {
	str := strings.ReplaceAll(hex, "#", "")
	str = strings.ReplaceAll(str, "0x", "")

	output, _ := strconv.ParseUint(str, 16, 32)
	return int(output)
}

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

// Returns items in listA but not in listB based on keyFunc.
func DifferenceBy[T any, K comparable](listA []T, listB []T, keyFn func(T) K) ([]T, map[K]struct{}) {
	seen := make(map[K]struct{}, len(listB))
	for _, v := range listB {
		seen[keyFn(v)] = struct{}{}
	}

	result := make([]T, 0)
	for _, v := range listA {
		if _, ok := seen[keyFn(v)]; !ok {
			result = append(result, v)
		}
	}

	return result, seen
}

// Length of a HEX colour without the '#' prefix.
// Since there are two digits per channel (#RRGGBB), we get a max of six.
const HEX_COLOUR_LEN = 6

// Validates whether str is a valid HEX colour string, independent of whether a '#' is already present.
func ValidateHexColour(str string) bool {
	str = strings.ReplaceAll(str, "#", "")
	if len(str) != HEX_COLOUR_LEN {
		return false
	}

	for i := range HEX_COLOUR_LEN {
		c := str[i] // current character in input string

		between09 := c >= '0' && c <= '9'
		betweenAF := c >= 'a' && c <= 'f'
		betweenAFUpper := c >= 'A' && c <= 'F'
		if !(between09 || betweenAF || betweenAFUpper) {
			return false
		}
	}

	return true
}

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

func ComparePtr[T cmp.Ordered](v1, v2 *T, defaultVal T) int {
	av, bv := defaultVal, defaultVal
	if v1 != nil {
		av = *v1
	}
	if v2 != nil {
		bv = *v2
	}

	return cmp.Compare(av, bv)
}

// Takes an input string and returns a slice containing each of the elements that were seperated by whitespace or sep.
//
// Similar to [strings.Fields] which splits elements by whitespace, we use [strings.FieldsFunc] to also
// check for commas, and any of the resulting empty strings elements are simply ignored.
// This should ensure it is able to handle most edge cases when the input is malformed.
//
// For example, the input ",foo1  , bar2,,, baz3" should produce the output: ["foo1" "bar2" "baz3"]
func ParseFieldsStr(input string, sep rune) ([]string, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == sep || unicode.IsSpace(r)
	})

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("failed to parse string list: no valid elements found")
	}

	return out, nil
}

func DefaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

// func DifferenceByReverse[T any, K comparable](listB []T, seenA map[K]struct{}, keyFn func(T) K) []T {
// 	onlyB := make([]T, 0)
// 	for _, v := range listB {
// 		if _, ok := seenA[keyFn(v)]; !ok {
// 			onlyB = append(onlyB, v)
// 		}
// 	}

// 	return onlyB
// }

// type UUID4 uuid.UUID
// func (u *UUID4) UnmarshalJSON(b []byte) error {
// 	id, err := uuid.Parse(string(b[:]))
// 	if err != nil {
// 		return err
// 	}
// 	*u = UUID4(id)
// 	return nil
// }

// func (u *UUID4) MarshalJSON() ([]byte, error) {
// 	return fmt.Appendf(nil, "\"%s\"", uuid.UUID(*u).String()), nil
// }
