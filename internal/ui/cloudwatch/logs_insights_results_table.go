package cloudwatch

import (
	"context"
	"strings"
	"sync"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

const LogRecordPtrCol = 0

type InsightsQueryResultsTable struct {
	*core.SelectableTable[string]
	rootView             core.View
	table                *tview.Table
	data                 [][]types.ResultField
	pollInterval         time.Duration
	mu                   sync.Mutex
	run                  *insightsRun
	status               string
	headingIdxMap        map[string]int
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
	ErrorMessageCallback func(text string, a ...any)
}

func NewInsightsQueryResultsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *InsightsQueryResultsTable {
	var selectableTable = core.NewSelectableTable[string]("Query Results", nil, serviceViewCtx.AppContext)

	var view = &InsightsQueryResultsTable{
		SelectableTable:      selectableTable,
		rootView:             selectableTable.Box,
		table:                selectableTable.GetTable(),
		data:                 nil,
		headingIdxMap:        map[string]int{},
		pollInterval:         defaultInsightsPollInterval,
		serviceCtx:           serviceViewCtx,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.HighlightSearch = true
	view.populateQueryResultsTable()
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.HelpView.View.
		AddItem("f", "Jump to next search result", nil).
		AddItem("F", "Jump to previous search result", nil)

	return view
}

func (inst *InsightsQueryResultsTable) populateQueryResultsTable() {
	inst.table.
		Clear().
		SetFixed(1, 0)

	var headingIdx = 0
	inst.headingIdxMap = map[string]int{}
	for rowIdx, rowData := range inst.data {
		var logStreamPtr = ""
		for _, resField := range rowData {
			if *resField.Field == "@ptr" {
				logStreamPtr = aws.ToString(resField.Value)
				break
			}
		}
		for _, resField := range rowData {
			if *resField.Field == "@ptr" {
				continue
			}

			var colIdx, ok = inst.headingIdxMap[*resField.Field]
			if !ok {
				inst.headingIdxMap[*resField.Field] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellText = aws.ToString(resField.Value)
			var newCell = core.NewTableCell[string](cellText, nil)

			if colIdx == LogRecordPtrCol {
				newCell = core.NewTableCell(cellText, &logStreamPtr)
			}

			inst.table.SetCell(rowIdx+1, colIdx, newCell)
		}
	}

	for heading, colIdx := range inst.headingIdxMap {
		core.SetTableHeading(inst.table, inst.serviceCtx.Theme, heading, colIdx)
	}

	inst.table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(inst.serviceCtx.Theme.MoreContrastBackgroundColor),
	)

	inst.SetTitleExtra(inst.status)
	inst.RefreshTitle(len(inst.data))

	inst.table.Select(1, 0)
	inst.table.ScrollToBeginning()
}

// insightsRun is a query started from the table
type insightsRun struct {
	id      string
	started time.Time
	cancel  context.CancelFunc
	// stopped is set once the query is finished or has been asked to stop
	stopped bool
	// cancelled is set once it has been asked to stop, after which what the
	// polling of it had in flight is not shown
	cancelled bool
}

// insightsStatusText says how far the query has got, e.g. "Running · 4s ·
// 1.2K matched · 10K scanned"
func insightsStatusText(status types.QueryStatus, elapsed time.Duration, stats *types.QueryStatistics) string {
	var parts = []string{string(status), elapsed.Round(time.Second).String()}
	if stats != nil {
		parts = append(parts,
			core.FormatCompactNumber(stats.RecordsMatched)+" matched",
			core.FormatCompactNumber(stats.RecordsScanned)+" scanned",
		)
	}
	return strings.Join(parts, " · ")
}

func (inst *InsightsQueryResultsTable) currentRun() *insightsRun {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.run
}

// showStatus shows how far the run has got, unless another query has been
// started since
func (inst *InsightsQueryResultsTable) showStatus(run *insightsRun, status string) {
	inst.serviceCtx.App.QueueUpdateDraw(func() {
		if inst.currentRun() != run || inst.isCancelled(run) {
			return
		}
		inst.setStatus(status)
	})
}

func (inst *InsightsQueryResultsTable) isCancelled(run *insightsRun) bool {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return run.cancelled
}

func (inst *InsightsQueryResultsTable) setStatus(status string) {
	inst.status = status
	inst.SetTitleExtra(status)
	inst.RefreshTitle(len(inst.data))
}

// showResults draws the results and the status of the run, unless another
// query has been started or it has been stopped since. The row that is
// selected is kept while it is still there.
func (inst *InsightsQueryResultsTable) showResults(run *insightsRun, status string, results [][]types.ResultField) {
	inst.serviceCtx.App.QueueUpdateDraw(func() {
		if inst.currentRun() != run || inst.isCancelled(run) {
			return
		}
		inst.status = status

		var row, column = inst.table.GetSelection()
		inst.data = results
		inst.populateQueryResultsTable()
		if row >= 1 && row <= len(results) {
			inst.table.Select(row, column)
		}
	})
}

// ExecuteQuery runs the query on the log groups, stopping the one that is
// running, and shows its results as they come in until it is done
func (inst *InsightsQueryResultsTable) ExecuteQuery(query InsightsQuery, logGroups []string) {
	if len(logGroups) == 0 {
		inst.ErrorMessageCallback("No log groups selected")
		return
	}

	var ctx, cancel = context.WithCancel(inst.serviceCtx.Ctx)
	var run = &insightsRun{started: time.Now(), cancel: cancel}

	inst.mu.Lock()
	var previous = inst.run
	inst.run = run
	inst.mu.Unlock()

	go func() {
		if previous != nil {
			inst.stopRun(previous)
		}
		inst.showStatus(run, "Starting")
		inst.runQuery(ctx, run, query, logGroups)
	}()
}

func (inst *InsightsQueryResultsTable) runQuery(
	ctx context.Context, run *insightsRun, query InsightsQuery, logGroups []string,
) {
	var api = inst.serviceCtx.Api
	var queryId, err = api.StartInsightsQuery(ctx, logGroups, query.StartTime, query.EndTime, query.Query)
	if err != nil {
		if ctx.Err() == nil {
			inst.showStatus(run, "Failed to start")
			inst.ErrorMessageCallback("%s", err.Error())
		}
		return
	}

	inst.mu.Lock()
	run.id = queryId
	inst.mu.Unlock()

	var elapsed = func() time.Duration { return time.Since(run.started) }
	var final awsapi.InsightsQueryResults
	final, err = pollInsightsQuery(ctx, api, queryId, inst.pollInterval, func(partial awsapi.InsightsQueryResults) {
		inst.showResults(run, insightsStatusText(partial.Status, elapsed(), partial.Statistics), partial.Results)
	})
	if ctx.Err() != nil {
		// Stopped or replaced, which is shown by whatever did that
		return
	}

	inst.mu.Lock()
	run.stopped = true
	inst.mu.Unlock()

	if err != nil {
		inst.showStatus(run, "Failed to get the results")
		inst.ErrorMessageCallback("%s", err.Error())
		return
	}

	inst.showResults(run, insightsStatusText(final.Status, elapsed(), final.Statistics), final.Results)
	if failure := insightsFailure(final.Status); len(failure) > 0 {
		inst.ErrorMessageCallback("%s", failure)
	}
}

// stopRun stops the polling of the run and the query itself if it is
// running, which it says it was
func (inst *InsightsQueryResultsTable) stopRun(run *insightsRun) bool {
	run.cancel()

	inst.mu.Lock()
	var wasRunning = !run.stopped
	run.stopped = true
	run.cancelled = true
	var queryId = run.id
	inst.mu.Unlock()

	if !wasRunning {
		return false
	}
	if len(queryId) > 0 {
		if _, err := inst.serviceCtx.Api.StopInsightsQuery(inst.serviceCtx.Ctx, queryId); err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	}
	return true
}

// StopQuery stops the query if one is running, keeping the results so far
func (inst *InsightsQueryResultsTable) StopQuery() {
	var run = inst.currentRun()
	if run == nil {
		return
	}

	go func() {
		if !inst.stopRun(run) {
			return
		}
		inst.serviceCtx.App.QueueUpdateDraw(func() {
			if inst.currentRun() == run {
				inst.setStatus(insightsStatusText(types.QueryStatusCancelled, time.Since(run.started), nil))
			}
		})
	}()
}

func (inst *InsightsQueryResultsTable) SetSelectedFunc(
	handler func(row int, column int),
) *InsightsQueryResultsTable {
	inst.table.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
	return inst
}

func (inst *InsightsQueryResultsTable) SetSelectionChangedFunc(
	handler func(row int, column int),
) *InsightsQueryResultsTable {
	inst.table.SetSelectionChangedFunc(handler)
	return inst
}

func (inst *InsightsQueryResultsTable) GetColumnCount() int {
	return inst.table.GetColumnCount()
}

func (inst *InsightsQueryResultsTable) GetRecordPtr(row int) string {
	return inst.SelectableTable.GetPrivateData(row, LogRecordPtrCol)
}

// To make the table data preview work (to be refactored)
func (inst *InsightsQueryResultsTable) GetPrivateData(row int, col int) string {
	return inst.GetCellText(row, col)
}
