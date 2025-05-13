package utils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/sanity-io/litter"
)

func ContainsNonAlphanumeric(input string) bool {
	// Define a regular expression pattern to match non-alphanumeric characters
	pattern := regexp.MustCompile(`[^a-zA-Z0-9]`)

	// If there are matches, it means non-alphanumeric characters were found
	return pattern.MatchString(input)
}

func Prettify(i any) string {
	litter.Config.StripPackageNames = true
	return litter.Sdump(i)
}

func HexToInt(hex string) int {
	str := strings.Replace(hex, "0x", "", -1)
	output, _ := strconv.ParseUint(str, 16, 32)

	return int(output)
}

func FormatTimestamp(unixTs float64) string {
	return strconv.FormatFloat(unixTs/1000, 'f', 0, 64)
}

func ParseJSON[T any](data []byte, result T) (T, error) {
	err := json.Unmarshal(data, &result)
	return result, err
}
