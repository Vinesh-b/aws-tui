package core

import (
	"strconv"
	"strings"
	"time"
)

// Values in details tables are formatted with these so they read the same
// across services. Unset values are empty, which details tables show as a
// placeholder.

// DetailsTimeFormat is the layout of times in details tables. They are shown
// in UTC as the api returns some times in local time and others in UTC.
const DetailsTimeFormat = time.DateTime

func FormatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(DetailsTimeFormat)
}

// FormatUnixMilli formats a time given in milliseconds since the epoch.
func FormatUnixMilli(ms *int64) string {
	if ms == nil {
		return ""
	}
	var t = time.UnixMilli(*ms)
	return FormatTime(&t)
}

// FormatUnixSeconds formats a time given as a string of seconds since the
// epoch. Text that isn't a number is empty.
func FormatUnixSeconds(seconds string) string {
	var value, err = strconv.ParseInt(seconds, 10, 64)
	if err != nil {
		return ""
	}
	var t = time.Unix(value, 0)
	return FormatTime(&t)
}

func FormatList[T ~string](items []T) string {
	var values = make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item))
	}
	return strings.Join(values, ", ")
}
