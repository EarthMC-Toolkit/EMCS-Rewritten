package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/sanity-io/litter"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type UUID4 uuid.UUID

func (u *UUID4) UnmarshalJSON(b []byte) error {
	id, err := uuid.Parse(string(b[:]))
	if err != nil {
		return err
	}
	*u = UUID4(id)
	return nil
}

func (u *UUID4) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"%s\"", uuid.UUID(*u).String()), nil
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
func CheckAlphanumeric(str string) string {
	return lo.Ternary(ContainsNonAlphanumeric(str), "", str)
}

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
	str := strings.ReplaceAll(hex, "0x", "")
	output, _ := strconv.ParseUint(str, 16, 32)

	return int(output)
}

// dis printer is bri ish
var printer = message.NewPrinter(language.BritishEnglish)

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
