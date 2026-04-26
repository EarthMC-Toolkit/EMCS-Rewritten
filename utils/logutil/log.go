package logutil

import (
	"fmt"
	"log"

	"github.com/sanity-io/litter"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	colour "github.com/fatih/color"
)

var (
	HIDDEN = colour.New(colour.FgWhite, colour.Concealed)
	WHITE  = colour.New(colour.Bold, colour.FgWhite)
	RED    = colour.New(colour.FgHiRed)
	GREEN  = colour.New(colour.FgGreen)
	YELLOW = colour.New(colour.FgYellow)
)

type Loggable interface {
	Log(args ...any)
}

// Attempts to prettify and log the value if the given error is nil, otherwise the error itself is logged.
func LogValOrErr(l Loggable, value any, err error) {
	if err == nil {
		l.Log(Prettify(value))
		return
	}

	l.Log(err)
}

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

func Printf(col *colour.Color, format string, args ...any) {
	fmt.Print(col.Sprintf(format, args...))
}

func Println(col *colour.Color, args ...any) {
	fmt.Println(col.Sprint(args...))
}

func Logf(col *colour.Color, format string, args ...any) {
	log.Print(col.Sprintf(format, args...))
}

func Logln(col *colour.Color, args ...any) {
	log.Println(col.Sprint(args...))
}
