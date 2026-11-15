package xray

import (
	"fmt"
	"math"
	"strconv"
)

// formatSeconds is a time in seconds as milliseconds when it is under a second
func formatSeconds(seconds *float64) string {
	if seconds == nil {
		return ""
	}
	if *seconds < 1 {
		return fmt.Sprintf("%dms", int(math.Round(*seconds*1000)))
	}
	return strconv.FormatFloat(*seconds, 'f', 2, 64) + "s"
}

func formatBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}
