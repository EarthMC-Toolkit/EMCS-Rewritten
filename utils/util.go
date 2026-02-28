package utils

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/samber/lo"
	"github.com/sanity-io/litter"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//const DateTimeFormat = "Jan 2 3PM MST"

// dis printer is bri ish
var printer = message.NewPrinter(language.BritishEnglish)

func PrettyPrint(v any) (int, error) {
	return printer.Print(Prettify(v))
}

// Calls Sprintf like usual, but in a humanized way. For example:
//
//	utils.HumanizedSprintf("Number is: %d\n", 10000)
//
// Outputs:
//
//	"Number is: 10,000"
func HumanizedSprintf(key message.Reference, a ...any) string {
	return printer.Sprintf(key, a...)
}

func HumanizeDuration(minutes float64) (float64, string) {
	if minutes >= 60 {
		return minutes / 60, "hr"
	}

	if minutes >= 1 {
		return minutes, "min"
	}

	return minutes * 60, "sec"
}

// Formats a time.Time to a string in the format "Wed, Jan 2nd 3PM UTC".
func FormatTime(t time.Time) string {
	t = t.UTC() // ensure UTC

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

func Prettify(v any) string {
	litter.Config.StripPackageNames = true
	return litter.Sdump(v)
}

type Loggable interface {
	Log(args ...any)
}

// Attempts to prettify and log the value if the given error is nil, otherwise the error itself is logged.
func CustomLog(l Loggable, value any, err error) {
	if err == nil {
		l.Log(Prettify(value))
		return
	}

	l.Log(err)
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

// MultiKeySort sorts arr in-place by multiple keys in order.
func MultiKeySort[T any](arr []T, keys []KeySortOption[T]) []T {
	slices.SortFunc(arr, func(a, b T) int {
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

	return arr
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

func MapValues[T any, R any](m map[string]T, fn func(string, T) R) []R {
	result := make([]R, 0, len(m))
	for k, v := range m {
		result = append(result, fn(k, v))
	}
	return result
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
