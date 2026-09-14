package logutil

import (
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

func Prettify(v any) string {
	litter.Config.StripPackageNames = true
	return litter.Sdump(v)
}

// Calls Sprintf like usual, but in a humanized way. For example:
//
//	logutil.HumanizedSprintf("Number is: %d\n", 10000)
//
// Outputs:
//
//	"Number is: 10,000"
func HumanizedSprintf(key message.Reference, a ...any) string {
	return printer.Sprintf(key, a...)
}
