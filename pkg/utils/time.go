package utils

import (
	"fmt"
	"time"
)

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
//	"10.52s"
//	"500.52ms"
func FormatElapsed(d time.Duration) string {
	d = d.Round(time.Millisecond)

	hrs := int64(d / time.Hour)
	mins := int64((d % time.Hour) / time.Minute)
	secs := int64((d % time.Minute) / time.Second)
	ms := float64(d%time.Second) / float64(time.Millisecond)

	if hrs > 0 {
		hrPostfix := "hrs"
		if hrs == 1 {
			hrPostfix = "hr"
		}
		return fmt.Sprintf("`%d%s`, `%dm` and `%ds`", hrs, hrPostfix, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("`%dm` and `%ds`", mins, secs)
	}
	if secs > 0 {
		return fmt.Sprintf("`%.2fs`", float64(secs)+ms/1000)
	}

	return fmt.Sprintf("`%.2fms`", ms)
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

	return fmt.Sprintf("%s %d%s %dAM UTC",
		//t.Weekday().String()[:3], // First three letters of the weekday word.
		t.Month().String()[:3], // First three letters of the month word.
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
