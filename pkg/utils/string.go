package utils

import (
	"fmt"
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

	output, _ := strconv.ParseUint(str, 16, 0) // err check not necessary. 0 is returned in all cases
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
