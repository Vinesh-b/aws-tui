package xray

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/gdamore/tcell/v2"
)

// requestStats are the statistics of the requests a service handled, or that
// one service made to another, which X-Ray gives in two types alike
type requestStats struct {
	total, ok, errors, throttles, faults int64
	// responseTime is the sum of the response times, in seconds
	responseTime float64
	// histogram is how many requests took each response time, in seconds
	histogram []types.HistogramEntry
}

func serviceStats(service types.Service) requestStats {
	var stats = requestStats{histogram: service.ResponseTimeHistogram}
	if summary := service.SummaryStatistics; summary != nil {
		stats.add(summary.TotalCount, summary.OkCount, summary.ErrorStatistics,
			summary.FaultStatistics, summary.TotalResponseTime)
	}
	return stats
}

func edgeStats(edge types.Edge) requestStats {
	var stats = requestStats{histogram: edge.ResponseTimeHistogram}
	if summary := edge.SummaryStatistics; summary != nil {
		stats.add(summary.TotalCount, summary.OkCount, summary.ErrorStatistics,
			summary.FaultStatistics, summary.TotalResponseTime)
	}
	return stats
}

func (inst *requestStats) add(
	total, ok *int64, errors *types.ErrorStatistics, faults *types.FaultStatistics, responseTime *float64,
) {
	inst.total = aws.ToInt64(total)
	inst.ok = aws.ToInt64(ok)
	if errors != nil {
		inst.errors = aws.ToInt64(errors.TotalCount)
		inst.throttles = aws.ToInt64(errors.ThrottleCount)
	}
	if faults != nil {
		inst.faults = aws.ToInt64(faults.TotalCount)
	}
	inst.responseTime = aws.ToFloat64(responseTime)
}

// colour is the colour of the worst of what went wrong with the requests, as
// a trace is coloured, and false when nothing did
func (inst requestStats) colour() (tcell.Color, bool) {
	switch {
	case inst.faults > 0:
		return traceFaultColour, true
	case inst.throttles > 0:
		return traceThrottleColour, true
	case inst.errors > 0:
		return traceErrorColour, true
	}
	return 0, false
}

// percentOf is the count as a percentage of the requests, which is empty
// without requests
func (inst requestStats) percentOf(count int64) string {
	if inst.total == 0 {
		return ""
	}
	return strconv.FormatFloat(float64(count)*100/float64(inst.total), 'f', 1, 64) + "%"
}

// average is the mean response time, which is empty without requests
func (inst requestStats) average() string {
	if inst.total == 0 {
		return ""
	}
	var average = inst.responseTime / float64(inst.total)
	return formatSeconds(&average)
}

// percentile is the response time the fraction of the requests took at most,
// such as 0.99 for p99, which is empty without a histogram
func (inst requestStats) percentile(fraction float64) string {
	var entries = slices.Clone(inst.histogram)
	slices.SortFunc(entries, func(a, b types.HistogramEntry) int { return cmp.Compare(a.Value, b.Value) })

	var count int64
	for _, entry := range entries {
		count += int64(entry.Count)
	}
	if count == 0 {
		return ""
	}

	var wanted = int64(math.Ceil(fraction * float64(count)))
	var seen int64
	for _, entry := range entries {
		seen += int64(entry.Count)
		if seen >= wanted {
			var value = entry.Value
			return formatSeconds(&value)
		}
	}
	return ""
}

// summary is the statistics in one line, for a call between services
func (inst requestStats) summary() string {
	var parts = []string{fmt.Sprintf("%d requests", inst.total)}
	if inst.total > 0 {
		parts = append(parts,
			inst.percentOf(inst.errors)+" errors",
			inst.percentOf(inst.faults)+" faults",
			"avg "+inst.average(),
		)
	}
	if p99 := inst.percentile(0.99); p99 != "" {
		parts = append(parts, "p99 "+p99)
	}
	return strings.Join(parts, ", ")
}

// serviceName is the name of a service, which clients don't have
func serviceName(service types.Service) string {
	if name := aws.ToString(service.Name); name != "" {
		return name
	}
	if aws.ToString(service.Type) == "client" {
		return "Clients"
	}
	return fmt.Sprintf("Service %d", aws.ToInt32(service.ReferenceId))
}
