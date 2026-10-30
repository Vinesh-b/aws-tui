package cloudwatch

import (
	"context"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/gdamore/tcell/v2"
)

const logMsgCol = 1

// defaultLogFollowInterval is the time between asking for the events that
// came after those shown, when following a log stream
const defaultLogFollowInterval = 2 * time.Second

// logFollowPageSize is the most events one ask gives. A full page means
// there are more waiting, so those are asked for without waiting.
const logFollowPageSize = 500

const logFollowingTitle = "following"

const logFilterOverlayId = "FILTER"

// logFilterView is the input for the filter pattern, which shows the pattern
// in use each time it is shown
type logFilterView struct {
	*core.FloatingSearchView
	pattern func() string
}

func (inst *logFilterView) OnShow() {
	inst.InputField.SetText(inst.pattern())
}

// logEventRanges are the time ranges to show the events of, ending now, which
// the range key goes through. The first has no span, and shows all the events.
var logEventRanges = append(
	[]insightsRange{{"All time", 0}},
	insightsRanges[:customInsightsRange]...,
)

type LogEventsTable struct {
	*core.SelectableTable[string]
	selectedLogGroup  string
	selectedLogStream string
	serviceCtx        *core.ServiceContext[awsapi.CloudWatchLogsApi]
	// followCancel stops following the stream, and is nil if it is not being
	// followed. It is only used on the UI goroutine.
	followCancel   context.CancelFunc
	followInterval time.Duration
	// rangeIdx is the range of logEventRanges the events are shown for, which
	// stays when another stream is opened
	rangeIdx int
	now      func() time.Time
	// filterPattern is what the events shown match, which is all if it is
	// empty. Like the range it stays when another stream is opened.
	filterPattern string
	filterView    *logFilterView
	// loads counts the reloads, so the events of one that was replaced by
	// another before they came are not shown
	loads int
}

func NewLogEventsTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogEventsTable {

	var view = &LogEventsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Log Events",
			core.TableRow{
				"Timestamp",
				"Message",
			},
			serviceContext.AppContext,
		),
		selectedLogGroup:  "",
		selectedLogStream: "",
		serviceCtx:        serviceContext,
		followInterval:    defaultLogFollowInterval,
		now:               time.Now,
	}

	view.HighlightSearch = true
	view.populateLogEventsTable(nil, false)
	view.SetLoadMoreFunc(func() {
		// Following brings the next events in, so nothing else pages them
		if view.IsFollowing() {
			view.MoreLoaded()
			return
		}
		view.RefreshLogEvents(false)
	})
	view.filterView = &logFilterView{
		FloatingSearchView: core.NewFloatingSearchView("Filter", 0, 3),
		pattern:            func() string { return view.filterPattern },
	}
	view.AddRuneToggleOverlay(logFilterOverlayId, view.filterView, core.APP_KEY_BINDINGS.Filter, false)
	view.filterView.InputField.SetDoneFunc(func(key tcell.Key) {
		if key == core.APP_KEY_BINDINGS.Done {
			view.ToggleOverlay(logFilterOverlayId, true)
			view.SetFilterPattern(view.filterView.InputField.GetText())
		}
	})
	view.HelpView.View.
		AddRune(core.APP_KEY_BINDINGS.Filter, "Show only the events that match a filter pattern").
		AddRune(core.APP_KEY_BINDINGS.Follow, "Follow the log stream as new events arrive").
		AddRune(core.APP_KEY_BINDINGS.TimeRange, "Show the events of the next time range")
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshLogEvents(true)
			return nil
		case core.APP_KEY_BINDINGS.Follow:
			view.ToggleFollow()
			return nil
		case core.APP_KEY_BINDINGS.TimeRange:
			view.NextTimeRange()
			return nil
		}
		return event
	})

	return view
}

func logEventRows(events []types.OutputLogEvent) ([]core.TableRow, []string) {
	var tableData = make([]core.TableRow, 0, len(events))
	var privateData = make([]string, 0, len(events))
	for _, row := range events {
		var message = aws.ToString(row.Message)
		tableData = append(tableData, core.TableRow{
			time.UnixMilli(aws.ToInt64(row.Timestamp)).Format("2006-01-02 15:04:05.000"),
			message,
		})
		privateData = append(privateData, message)
	}
	return tableData, privateData
}

func (inst *LogEventsTable) populateLogEventsTable(events []types.OutputLogEvent, reset bool) {
	var tableData, privateData = logEventRows(events)

	if !reset {
		inst.ExtendData(tableData, privateData)
		return
	}

	inst.SetData(tableData, privateData, logMsgCol)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *LogEventsTable) RefreshLogEvents(reset bool) {
	// The events are read again from the start, so the ones followed so far
	// would come after the wrong ones
	if reset {
		inst.StopFollow()
		inst.loads++
	}

	var load = inst.loads
	var startTime = inst.rangeStart()
	var filterPattern = inst.filterPattern
	var events []types.OutputLogEvent
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		if len(filterPattern) > 0 {
			events, err = inst.serviceCtx.Api.ListFilteredLogEvents(
				inst.serviceCtx.Ctx,
				inst.selectedLogGroup,
				inst.selectedLogStream,
				filterPattern,
				startTime,
				reset,
			)
		} else {
			events, err = inst.serviceCtx.Api.ListLogEvents(
				inst.serviceCtx.Ctx,
				inst.selectedLogGroup,
				inst.selectedLogStream,
				startTime,
				reset,
			)
		}
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if load == inst.loads {
			inst.populateLogEventsTable(events, reset)
		}
	})
}

func (inst *LogEventsTable) SetSelectedLogGroup(logGroup string) {
	inst.selectedLogGroup = logGroup
}

func (inst *LogEventsTable) SetSelectedLogStream(logStream string) {
	inst.StopFollow()
	inst.selectedLogStream = logStream
	inst.showTitleExtra()
}

func (inst *LogEventsTable) showTitleExtra() {
	var parts = []string{inst.selectedLogStream}
	if inst.rangeIdx > 0 {
		parts = append(parts, logEventRanges[inst.rangeIdx].label)
	}
	if len(inst.filterPattern) > 0 {
		parts = append(parts, "filter: "+inst.filterPattern)
	}
	if inst.IsFollowing() {
		parts = append(parts, logFollowingTitle)
	}
	inst.SetTitleExtra(strings.Join(parts, " · "))
	inst.RefreshTitle(0)
}

// SetFilterPattern loads the events that match the filter pattern, or all of
// them if it is empty. They are searched for on the server, so it finds those
// that are not loaded yet.
func (inst *LogEventsTable) SetFilterPattern(pattern string) {
	if len(inst.selectedLogGroup) == 0 || len(inst.selectedLogStream) == 0 {
		return
	}

	inst.filterPattern = pattern
	inst.RefreshLogEvents(true)
	inst.showTitleExtra()
}

// rangeStart is when the range being shown starts, which is the zero time if
// it is all time
func (inst *LogEventsTable) rangeStart() time.Time {
	var span = logEventRanges[inst.rangeIdx].span
	if span == 0 {
		return time.Time{}
	}
	return inst.now().Add(-span)
}

// NextTimeRange loads the events of the next time range, going back to all
// time after the last. The range is counted back from now.
func (inst *LogEventsTable) NextTimeRange() {
	if len(inst.selectedLogGroup) == 0 || len(inst.selectedLogStream) == 0 {
		return
	}

	inst.rangeIdx = (inst.rangeIdx + 1) % len(logEventRanges)
	inst.RefreshLogEvents(true)
	inst.showTitleExtra()
}

func (inst *LogEventsTable) IsFollowing() bool {
	return inst.followCancel != nil
}

// ToggleFollow starts following the log stream, which adds the events that
// come after those shown as they arrive, or stops following it
func (inst *LogEventsTable) ToggleFollow() {
	if inst.IsFollowing() {
		inst.StopFollow()
		return
	}
	if len(inst.selectedLogGroup) == 0 || len(inst.selectedLogStream) == 0 {
		return
	}
	// The events that come after are not filtered
	if len(inst.filterPattern) > 0 {
		inst.ErrorMessageCallback("Clear the filter to follow the log stream")
		return
	}

	var ctx, cancel = context.WithCancel(inst.serviceCtx.Ctx)
	inst.followCancel = cancel
	inst.showTitleExtra()
	go inst.follow(ctx, inst.selectedLogGroup, inst.selectedLogStream)
}

// StopFollow stops following the log stream, if it is being followed. It is
// used on the UI goroutine.
func (inst *LogEventsTable) StopFollow() {
	if inst.followCancel == nil {
		return
	}
	inst.followCancel()
	inst.followCancel = nil
	inst.showTitleExtra()
}

func (inst *LogEventsTable) follow(ctx context.Context, logGroup string, logStream string) {
	for {
		var events, err = inst.serviceCtx.Api.FollowLogEvents(ctx, logGroup, logStream)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
			inst.serviceCtx.App.QueueUpdateDraw(func() {
				if ctx.Err() == nil {
					inst.StopFollow()
				}
			})
			return
		}

		if len(events) > 0 {
			inst.serviceCtx.App.QueueUpdateDraw(func() {
				if ctx.Err() == nil {
					inst.appendFollowed(events)
				}
			})
		}

		if len(events) >= logFollowPageSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(inst.followInterval):
		}
	}
}

// appendFollowed adds the events to the end of the table. The cursor goes
// with the end if it was there, and otherwise stays so they can be read.
func (inst *LogEventsTable) appendFollowed(events []types.OutputLogEvent) {
	var table = inst.GetTable()
	var row, _ = table.GetSelection()
	var atEnd = row >= table.GetRowCount()-1

	var tableData, privateData = logEventRows(events)
	inst.ExtendData(tableData, privateData)

	// ExtendData leaves the cursor on the row that was last
	if atEnd {
		inst.Select(table.GetRowCount()-1, 0)
	} else {
		inst.Select(row, 0)
	}
}

func (inst *LogEventsTable) GetFullLogMessage(row int) string {
	if row < 1 {
		return ""
	}
	return inst.GetPrivateData(row, logMsgCol)
}
