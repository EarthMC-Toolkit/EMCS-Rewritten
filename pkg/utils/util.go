package utils

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/samber/lo"
)

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

// Takes a Hexadecimal string (# and 0x prefixes allowed) and parses it into a Decimal format (integer)
// which uses the base 10 number system; not to be confused with a float which allows decimal points.
func HexToInt(hex string) int {
	str := strings.ReplaceAll(hex, "#", "")
	str = strings.ReplaceAll(str, "0x", "")

	output, _ := strconv.ParseUint(str, 16, 32) // err check not necessary. 0 is returned in all cases
	return int(output)
}

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

func CmpPtrDefault[T cmp.Ordered](v1, v2 *T, defaultVal T) int {
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
