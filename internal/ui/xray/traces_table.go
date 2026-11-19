package xray

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"

	"github.com/gdamore/tcell/v2"
)

// traceRange is a time range of the traces, which ends when they are loaded.
// Custom has no span, and is between the times that were entered.
type traceRange struct {
	label string
	span  time.Duration
}

// traceRanges are the time ranges to pick from, which the range key goes
// through, but for Custom. X-Ray finds traces by the time they started, so the
// longest is kept to a few hours.
var traceRanges = []traceRange{
	{"Last 5 minutes", 5 * time.Minute},
	{"Last 15 minutes", 15 * time.Minute},
	{"Last hour", time.Hour},
	{"Last 3 hours", 3 * time.Hour},
	{"Last 6 hours", 6 * time.Hour},
	{"Custom", 0},
}

// The colours of the traces that had a fault (5xx), were throttled (429) or
// had another error (4xx). A throttled trace also has an error, so throttling
// is checked first, and a fault outranks both.
const (
	traceFaultColour    = tcell.ColorRed
	traceThrottleColour = tcell.ColorFuchsia
	traceErrorColour    = tcell.ColorYellow
)

// traceColour is the colour of a trace by the worst thing that happened in it,
// and false for one that had none of them
func traceColour(summary types.TraceSummary) (tcell.Color, bool) {
	switch {
	case aws.ToBool(summary.HasFault):
		return traceFaultColour, true
	case aws.ToBool(summary.HasThrottle):
		return traceThrottleColour, true
	case aws.ToBool(summary.HasError):
		return traceErrorColour, true
	}
	return 0, false
}

const (
	defaultTraceRangeIdx = 2
	customTraceRangeIdx  = 5
)

type TracesTable struct {
	*core.SelectableTable[types.TraceSummary]
	selectedTrace types.TraceSummary
	serviceCtx    *core.ServiceContext[awsapi.XRayApi]
	// rangeIdx is the range of traceRanges the traces are shown for, and
	// customStart and customEnd the times of Custom
	rangeIdx    int
	customStart time.Time
	customEnd   time.Time
	// filter is the X-Ray filter expression the traces are found with, which
	// is empty for all of them
	filter string
	now    func() time.Time
	// rangeChanged is told the range and its times when they change, and when
	// a range ending now is loaded again
	rangeChanged func(rangeIdx int, start, end time.Time)
	// queryLogs is called to query the logs of the selected trace
	queryLogs func()
	// loaded are the traces loaded so far, all pages of them, and
	// loadedChanged is called when they change
	loaded        []types.TraceSummary
	loadedChanged func()
	// loads counts the reloads, so the traces of one that was replaced by
	// another before they came are not shown
	loads int
}

func NewTracesTable(
	serviceContext *core.ServiceContext[awsapi.XRayApi],
) *TracesTable {
	var table = &TracesTable{
		SelectableTable: core.NewSelectableTable[types.TraceSummary](
			"Traces",
			core.TableRow{
				"Start Time",
				"Entry Point",
				"Method",
				"Status",
				"Response Time",
				"URL",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(5, 1).SetFilterColumns(1, 2, 3, 5),
		serviceCtx: serviceContext,
		rangeIdx:   defaultTraceRangeIdx,
		now:        time.Now,
		queryLogs:  func() {},

		loadedChanged: func() {},
	}

	table.HighlightSearch = true
	table.SetRowColourFunc(func(_ core.TableRow, summary types.TraceSummary) (tcell.Color, bool) {
		return traceColour(summary)
	})
	table.showRange(table.window())
	table.HelpView.View.
		AddRune(core.APP_KEY_BINDINGS.TimeRange, "Show the traces of the next time range").
		AddRune(core.APP_KEY_BINDINGS.TableQuery, "Query the logs of the trace with Logs Insights")
	table.populateTracesTable(nil, true)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetLoadMoreFunc(func() { table.RefreshTraces(false) })

	return table
}

func entryPointName(summary types.TraceSummary) string {
	if summary.EntryPoint == nil {
		return ""
	}
	return aws.ToString(summary.EntryPoint.Name)
}

func traceHttp(summary types.TraceSummary) types.Http {
	if summary.Http == nil {
		return types.Http{}
	}
	return *summary.Http
}

func (inst *TracesTable) populateTracesTable(summaries []types.TraceSummary, reset bool) {
	var tableData []core.TableRow
	for _, summary := range summaries {
		var http = traceHttp(summary)
		tableData = append(tableData, core.TableRow{
			core.FormatTime(summary.StartTime),
			entryPointName(summary),
			aws.ToString(http.HttpMethod),
			core.FormatOptionalInt(http.HttpStatus),
			formatSeconds(summary.ResponseTime),
			aws.ToString(http.HttpURL),
		})
	}

	if !reset {
		inst.loaded = append(inst.loaded, summaries...)
		inst.ExtendData(tableData, summaries)
		inst.loadedChanged()
		return
	}

	// Nothing is selected in an empty table
	if len(summaries) == 0 {
		inst.selectedTrace = types.TraceSummary{}
	}
	inst.loaded = slices.Clone(summaries)
	inst.SetData(tableData, summaries, 0)
	inst.Select(1, 0)
	inst.loadedChanged()
}

// SetLoadedChangedFunc sets what is called when the traces loaded change,
// which is when they are loaded again and when another page of them comes
func (inst *TracesTable) SetLoadedChangedFunc(changed func()) {
	inst.loadedChanged = changed
}

// LoadedTraces are the traces loaded so far, of all the pages loaded
func (inst *TracesTable) LoadedTraces() []types.TraceSummary {
	return slices.Clone(inst.loaded)
}

// RangeLabel is the name of the time range the traces are of
func (inst *TracesTable) RangeLabel() string {
	return traceRanges[inst.rangeIdx].label
}

// FilterExpression is the X-Ray filter expression the traces are found with
func (inst *TracesTable) FilterExpression() string {
	return inst.filter
}

// RefreshTraces loads the newest traces of the time range again, or the page
// after those shown when reset is false
func (inst *TracesTable) RefreshTraces(reset bool) {
	if reset {
		inst.loads++
	}
	var load = inst.loads
	var start, end = inst.window()
	if reset {
		inst.showRange(start, end)
	}
	// The filter is read here, as it is changed on the UI goroutine while
	// the traces load on another
	var filter = inst.filter
	var summaries []types.TraceSummary
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		summaries, err = inst.serviceCtx.Api.GetTraceSummaries(ctx, start, end, filter, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		// The traces were loaded again while these were loading
		if load != inst.loads {
			return
		}
		inst.populateTracesTable(summaries, reset)
	})
}

// window is the times of the range, which for all but Custom end now
func (inst *TracesTable) window() (start, end time.Time) {
	if inst.rangeIdx == customTraceRangeIdx {
		return inst.customStart, inst.customEnd
	}
	end = inst.now()
	return end.Add(-traceRanges[inst.rangeIdx].span), end
}

// NextTimeRange loads the traces of the next range ending now, going back to
// the shortest after the longest, and from Custom
func (inst *TracesTable) NextTimeRange() {
	if inst.rangeIdx == customTraceRangeIdx {
		inst.SetTimeRange(0)
		return
	}
	inst.SetTimeRange((inst.rangeIdx + 1) % customTraceRangeIdx)
}

// SetTimeRange loads the traces of the range of traceRanges, which is counted
// back from now. Custom is only picked by entering its times.
func (inst *TracesTable) SetTimeRange(rangeIdx int) {
	if rangeIdx < 0 || rangeIdx >= customTraceRangeIdx {
		return
	}
	inst.rangeIdx = rangeIdx
	inst.RefreshTraces(true)
}

// SetCustomRange loads the traces that started between the times, which is an
// error if the end is not after the start
func (inst *TracesTable) SetCustomRange(start, end time.Time) error {
	if !end.After(start) {
		return errors.New("The end time is not after the start time")
	}
	inst.rangeIdx = customTraceRangeIdx
	inst.customStart, inst.customEnd = start, end
	inst.RefreshTraces(true)
	return nil
}

// SetFilterExpression loads the traces of the range that match the X-Ray
// filter expression, or all of them when it is empty. The filter is kept for
// the ranges picked after it.
func (inst *TracesTable) SetFilterExpression(filter string) {
	inst.filter = strings.TrimSpace(filter)
	inst.RefreshTraces(true)
}

// SetRangeChangedFunc is told the range and its times when they change, and
// straight away of those shown now
func (inst *TracesTable) SetRangeChangedFunc(changed func(rangeIdx int, start, end time.Time)) {
	inst.rangeChanged = changed
	var start, end = inst.window()
	changed(inst.rangeIdx, start, end)
}

func (inst *TracesTable) showRange(start, end time.Time) {
	inst.SetTitleExtra(traceRanges[inst.rangeIdx].label)
	inst.RefreshTitle(0)
	if inst.rangeChanged != nil {
		inst.rangeChanged(inst.rangeIdx, start, end)
	}
}

func (inst *TracesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTrace = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *TracesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshTraces(true)
			return nil
		case core.APP_KEY_BINDINGS.TimeRange:
			inst.NextTimeRange()
			return nil
		case core.APP_KEY_BINDINGS.TableQuery:
			inst.queryLogs()
			return nil
		}
		return capture(event)
	})
}

// SetQueryLogsFunc sets what queries the logs of the selected trace
func (inst *TracesTable) SetQueryLogsFunc(queryLogs func()) {
	inst.queryLogs = queryLogs
}

func (inst *TracesTable) GetSelectedTrace() types.TraceSummary {
	return inst.selectedTrace
}
