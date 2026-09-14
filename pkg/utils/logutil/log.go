package logutil

import (
	"fmt"
	"log"
	"os"
	"strings"

	colour "github.com/fatih/color"
)

var FileLog *log.Logger
var DebugLogEnabled = false

var (
	FAINT  = colour.New(colour.FgWhite, colour.Concealed) // DEBUG
	WHITE  = colour.New(colour.Bold, colour.FgWhite)      // DEFAULT/NORMAL
	BLUE   = colour.New(colour.FgHiBlue)                  // INFO/OPERATIONAL (Foreground)
	BLUEBG = colour.New(colour.BgBlue, colour.FgHiWhite)  // INFO/OPERATIONAL (Background)
	GREEN  = colour.New(colour.FgGreen)                   // SUCCESS
	YELLOW = colour.New(colour.FgYellow)                  // WARN
	RED    = colour.New(colour.FgHiRed)                   // ERROR (Foreground)
	REDBG  = colour.New(colour.BgRed, colour.FgHiWhite)   // ERROR (Background)
)

func InitFile(fpath string) error {
	file, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	FileLog = log.New(file, "", log.Ldate|log.Ltime|log.LUTC)
	return nil
}

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

func Logf(col *colour.Color, format string, args ...any) {
	if strings.HasPrefix(format, "DEBUG") && !DebugLogEnabled {
		return
	}

	log.Print(col.Sprintf(format, args...))
}

func Logln(col *colour.Color, args ...any) {
	if strings.HasPrefix(fmt.Sprint(args...), "DEBUG") && !DebugLogEnabled {
		return
	}

	log.Println(col.Sprint(args...))
}

func Printf(col *colour.Color, format string, args ...any) {
	if strings.HasPrefix(format, "DEBUG") && !DebugLogEnabled {
		return
	}

	fmt.Print(col.Sprintf(format, args...))
}

func Println(col *colour.Color, args ...any) {
	if strings.HasPrefix(fmt.Sprint(args...), "DEBUG") && !DebugLogEnabled {
		return
	}

	fmt.Println(col.Sprint(args...))
}

func Space() {
	fmt.Println()
}
