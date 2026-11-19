package xray

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/gdamore/tcell/v2"
)

// slowestTracesShown is how many of the slowest traces of a group are listed
const slowestTracesShown = 5

// traceGroup is the traces that came in at the same entry point for the same
// method and URL, with the statistics of their requests
type traceGroup struct {
	entryPoint, method, url string
	traces                  []types.TraceSummary
	stats                   requestStats
}

// groupTraces groups the traces by entry point, method and URL, the groups
// with the most traces first
func groupTraces(summaries []types.TraceSummary) []traceGroup {
	type key struct{ entryPoint, method, url string }
	var groups = map[key]*traceGroup{}
	var order []key

	for _, summary := range summaries {
		var http = traceHttp(summary)
		var k = key{entryPointName(summary), aws.ToString(http.HttpMethod), aws.ToString(http.HttpURL)}
		var group, ok = groups[k]
		if !ok {
			group = &traceGroup{entryPoint: k.entryPoint, method: k.method, url: k.url}
			groups[k] = group
			order = append(order, k)
		}
		group.traces = append(group.traces, summary)
	}

	var result []traceGroup
	for _, k := range order {
		var group = groups[k]
		group.stats = traceStats(group.traces)
		result = append(result, *group)
	}
	slices.SortStableFunc(result, func(a, b traceGroup) int {
		if byCount := cmp.Compare(len(b.traces), len(a.traces)); byCount != 0 {
			return byCount
		}
		return cmp.Or(
			strings.Compare(a.entryPoint, b.entryPoint),
			strings.Compare(a.url, b.url),
			strings.Compare(a.method, b.method),
		)
	})
	return result
}

// traceStats are the statistics of the requests of the traces, as those of a
// service, with a histogram of their response times
func traceStats(summaries []types.TraceSummary) requestStats {
	var stats = requestStats{total: int64(len(summaries))}
	for _, summary := range summaries {
		var failed = false
		if aws.ToBool(summary.HasFault) {
			stats.faults++
			failed = true
		}
		if aws.ToBool(summary.HasError) {
			stats.errors++
			failed = true
		}
		if aws.ToBool(summary.HasThrottle) {
			stats.throttles++
		}
		if !failed {
			stats.ok++
		}
		if summary.ResponseTime != nil {
			stats.responseTime += *summary.ResponseTime
			stats.histogram = append(stats.histogram, types.HistogramEntry{Value: *summary.ResponseTime, Count: 1})
		}
	}
	return stats
}

// filterQuote is the text as a string of an X-Ray filter expression
func filterQuote(text string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(text, `\`, `\\`), `"`, `\"`) + `"`
}

// filterExpression is the X-Ray filter expression that finds the traces of
// the group: those of its URL and method, or those through its entry point
// when they have no URL
func (inst traceGroup) filterExpression() (string, error) {
	if inst.url != "" {
		var expression = "http.url = " + filterQuote(inst.url)
		if inst.method != "" {
			expression += " AND http.method = " + filterQuote(inst.method)
		}
		return expression, nil
	}
	if inst.entryPoint != "" {
		return "service(" + filterQuote(inst.entryPoint) + ")", nil
	}
	return "", errors.New("The traces of this group have no URL or entry point to find them by")
}

// TraceAnalyticsTable groups the traces the traces table has loaded, of its
// range and filter, to show which requests are slow or fail
type TraceAnalyticsTable struct {
	*core.SelectableTable[traceGroup]
	traces        *TracesTable
	groups        []traceGroup
	selectedGroup traceGroup
}

func NewTraceAnalyticsTable(traces *TracesTable, appCtx *core.AppContext) *TraceAnalyticsTable {
	var table = &TraceAnalyticsTable{
		SelectableTable: core.NewSelectableTable[traceGroup](
			"Trace Analytics",
			core.TableRow{
				"Entry Point",
				"Method",
				"URL",
				"Traces",
				"Errors",
				"Faults",
				"Average",
				"p50",
				"p99",
			},
			appCtx,
		).SetColumnExpansion(2, 1).SetFilterColumns(0, 1, 2),
		traces: traces,
	}

	table.HelpView.View.
		AddRune(core.APP_KEY_BINDINGS.TimeRange, "Load the traces of the next time range").
		AddKey(core.APP_KEY_BINDINGS.Done, "Show the traces of the group")
	table.SetRowColourFunc(func(_ core.TableRow, group traceGroup) (tcell.Color, bool) {
		return group.stats.colour()
	})
	table.populateAnalyticsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

// Refresh groups the traces that are loaded now
func (inst *TraceAnalyticsTable) Refresh() {
	inst.groups = groupTraces(inst.traces.LoadedTraces())
	inst.populateAnalyticsTable()
}

func (inst *TraceAnalyticsTable) populateAnalyticsTable() {
	var tableData []core.TableRow
	var count = 0
	for _, group := range inst.groups {
		var stats = group.stats
		count += len(group.traces)
		tableData = append(tableData, core.TableRow{
			group.entryPoint,
			group.method,
			group.url,
			strconv.Itoa(len(group.traces)),
			stats.percentOf(stats.errors),
			stats.percentOf(stats.faults),
			stats.average(),
			stats.percentile(0.5),
			stats.percentile(0.99),
		})
	}

	// The groups are only of the traces loaded, which the title tells
	var extra = fmt.Sprintf("%s, %d traces", inst.traces.RangeLabel(), count)
	if inst.traces.FilterExpression() != "" {
		extra += ", filtered"
	}
	inst.SetTitleExtra(extra)

	// Nothing is selected in an empty table
	if len(inst.groups) == 0 {
		inst.selectedGroup = traceGroup{}
	}
	inst.SetData(tableData, inst.groups, 0)
	inst.Select(1, 0)
}

func (inst *TraceAnalyticsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedGroup = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *TraceAnalyticsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.traces.RefreshTraces(true)
			return nil
		case core.APP_KEY_BINDINGS.TimeRange:
			inst.traces.NextTimeRange()
			return nil
		}
		return capture(event)
	})
}

func (inst *TraceAnalyticsTable) GetSelectedGroup() traceGroup {
	return inst.selectedGroup
}

// TraceGroupDetailsTable shows a group of traces with the statistics of their
// requests, their statuses and the slowest of them
type TraceGroupDetailsTable struct {
	*core.DetailsTable
	group traceGroup
}

func NewTraceGroupDetailsTable(appCtx *core.AppContext) *TraceGroupDetailsTable {
	var table = &TraceGroupDetailsTable{
		DetailsTable: core.NewDetailsTable("Group Details", appCtx),
	}
	table.populateDetailsTable()
	return table
}

func (inst *TraceGroupDetailsTable) SetGroup(group traceGroup) {
	inst.group = group
	inst.populateDetailsTable()
}

// statusRows are how many traces ended with each HTTP status, in order
func statusRows(traces []types.TraceSummary) []core.TableRow {
	var counts = map[string]int{}
	for _, summary := range traces {
		var status = core.FormatOptionalInt(traceHttp(summary).HttpStatus)
		if status == "" {
			status = "None"
		}
		counts[status]++
	}
	var rows []core.TableRow
	for _, status := range slices.Sorted(maps.Keys(counts)) {
		rows = append(rows, core.TableRow{status, strconv.Itoa(counts[status])})
	}
	return rows
}

// slowestRows are the slowest of the traces, by their response time
func slowestRows(traces []types.TraceSummary) []core.TableRow {
	var timed []types.TraceSummary
	for _, summary := range traces {
		if summary.ResponseTime != nil {
			timed = append(timed, summary)
		}
	}
	slices.SortStableFunc(timed, func(a, b types.TraceSummary) int {
		return cmp.Compare(*b.ResponseTime, *a.ResponseTime)
	})

	var rows []core.TableRow
	for _, summary := range timed[:min(len(timed), slowestTracesShown)] {
		rows = append(rows, core.TableRow{
			aws.ToString(summary.Id),
			formatSeconds(summary.ResponseTime) + ", " + core.FormatTime(summary.StartTime),
		})
	}
	return rows
}

func (inst *TraceGroupDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var group = inst.group

	if len(group.traces) > 0 {
		var stats = group.stats
		tableData = []core.TableRow{
			{"Entry Point", group.entryPoint},
			{"Method", group.method},
			{"URL", group.url},
			{"Traces", strconv.FormatInt(stats.total, 10)},
			{"OK", strconv.FormatInt(stats.ok, 10)},
			{"Errors", countAndPercent(stats, stats.errors)},
			{"Throttles", countAndPercent(stats, stats.throttles)},
			{"Faults", countAndPercent(stats, stats.faults)},
			{"Average", stats.average()},
			{"p50", stats.percentile(0.5)},
			{"p90", stats.percentile(0.9)},
			{"p99", stats.percentile(0.99)},
			{"Max", stats.percentile(1)},
		}
		tableData = append(tableData, core.DetailsSection("Statuses", statusRows(group.traces)...)...)
		tableData = append(tableData, core.DetailsSection("Slowest Traces", slowestRows(group.traces)...)...)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}
