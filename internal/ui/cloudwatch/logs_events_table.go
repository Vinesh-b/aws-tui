package cloudwatch

import (
	"context"
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

type LogEventsTable struct {
	*core.SelectableTable[string]
	data              []types.OutputLogEvent
	selectedLogGroup  string
	selectedLogStream string
	serviceCtx        *core.ServiceContext[awsapi.CloudWatchLogsApi]
	// followCancel stops following the stream, and is nil if it is not being
	// followed. It is only used on the UI goroutine.
	followCancel   context.CancelFunc
	followInterval time.Duration
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
		data:              nil,
		selectedLogGroup:  "",
		selectedLogStream: "",
		serviceCtx:        serviceContext,
		followInterval:    defaultLogFollowInterval,
	}

	view.HighlightSearch = true
	view.populateLogEventsTable(false)
	view.SetLoadMoreFunc(func() {
		// Following brings the next events in, so nothing else pages them
		if view.IsFollowing() {
			view.MoreLoaded()
			return
		}
		view.RefreshLogEvents(false)
	})
	view.HelpView.View.AddRune(core.APP_KEY_BINDINGS.Follow, "Follow the log stream as new events arrive")
	view.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshLogEvents(true)
			return nil
		case core.APP_KEY_BINDINGS.Follow:
			view.ToggleFollow()
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

func (inst *LogEventsTable) populateLogEventsTable(reset bool) {
	var tableData, privateData = logEventRows(inst.data)

	// The table now owns the text, don't keep the raw events alive as well
	inst.data = nil

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
	}

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.serviceCtx.Api.ListLogEvents(
			inst.serviceCtx.Ctx,
			inst.selectedLogGroup,
			inst.selectedLogStream,
			reset,
		)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLogEventsTable(reset)
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
	var extra = inst.selectedLogStream
	if inst.IsFollowing() {
		extra += " " + logFollowingTitle
	}
	inst.SetTitleExtra(extra)
	inst.RefreshTitle(0)
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
