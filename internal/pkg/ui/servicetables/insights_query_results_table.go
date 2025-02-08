package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

type InsightsQueryRunner struct {
	data    [][]types.ResultField
	queryId string

	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
	ErrorMessageCallback func(text string, a ...any)
}

func NewInsightsQueryRunner(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
) *InsightsQueryRunner {

	return &InsightsQueryRunner{
		data:                 nil,
		queryId:              "",
		app:                  app,
		api:                  api,
		ErrorMessageCallback: func(text string, a ...any) {},
	}
}

func (inst *InsightsQueryRunner) ExecuteInsightsQuery(
	query InsightsQuery, logGroups []string, resultChan chan [][]types.ResultField,
) {
	if len(logGroups) == 0 {
		inst.ErrorMessageCallback("No log groups selected")
		return
	}

	go func() {
		if len(inst.queryId) > 0 {
			var _, err = inst.api.StopInightsQuery(inst.queryId)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
			inst.queryId = ""
		}

		var err error = nil
		inst.queryId, err = inst.api.StartInightsQuery(
			logGroups,
			query.startTime,
			query.endTime,
			query.query,
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		var results [][]types.ResultField
		var status types.QueryStatus
		for range 10 {
			if len(inst.queryId) == 0 {
				break
			}

			results, status, err = inst.api.GetInightsQueryResults(inst.queryId)

			switch status {
			case types.QueryStatusRunning, types.QueryStatusScheduled:
				time.Sleep(2 * time.Second)
			case types.QueryStatusComplete, types.QueryStatusCancelled:
				inst.queryId = ""
				break
			default:
				inst.queryId = ""
				if err != nil {
					inst.ErrorMessageCallback(err.Error())
				} else {
					inst.ErrorMessageCallback(fmt.Sprintf("Query failed with status %s", status))
				}
				break
			}
		}

		resultChan <- results
	}()
}

const LogRecordPtrCol = 0

type InsightsQueryResultsTable struct {
	*core.SelectableTable[string]
	queryView            *FloatingInsightsQueryInputView
	rootView             core.View
	table                *tview.Table
	data                 [][]types.ResultField
	queryId              string
	selectedLogGroups    []string
	headingIdxMap        map[string]int
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchLogsApi]
	ErrorMessageCallback func(text string, a ...any)
}

func NewInsightsQueryResultsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *InsightsQueryResultsTable {
	var selectableTable = core.NewSelectableTable[string]("Query Results", nil, serviceViewCtx.AppContext)
	var queryView = NewFloatingInsightsQueryInputView(serviceViewCtx.AppContext)
	selectableTable.AddRuneToggleOverlay("QUERY", queryView, core.APP_KEY_BINDINGS.TableQuery, false)

	var view = &InsightsQueryResultsTable{
		SelectableTable:      selectableTable,
		queryView:            queryView,
		rootView:             selectableTable.Box,
		table:                selectableTable.GetTable(),
		data:                 nil,
		queryId:              "",
		selectedLogGroups:    nil,
		headingIdxMap:        map[string]int{},
		serviceCtx:           serviceViewCtx,
		ErrorMessageCallback: func(text string, a ...any) {},
	}

	view.HighlightSearch = true
	view.populateQueryResultsTable()
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshResults()
		}
		return event
	})

	view.queryView.Input.DoneButton.SetSelectedFunc(func() {
		view.ExecuteQuery()
	})

	view.queryView.Input.CancelButton.SetSelectedFunc(func() {
		view.StopQuery()
	})

	view.HelpView.View.
		AddItem("f", "Jump to next search result", nil).
		AddItem("F", "Jump to previous search result", nil).
		AddItem("q", "To show query view", func() {
			view.ToggleOverlay("QUERY", false)
		})

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
		core.SetTableHeading(inst.table, heading, colIdx)
	}

	inst.table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(core.MoreContrastBackgroundColor),
	)

	inst.RefreshTitle(len(inst.data))

	inst.table.Select(1, 0)
	inst.table.ScrollToBeginning()
}

func (inst *InsightsQueryResultsTable) RefreshResults() {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var results [][]types.ResultField
		var status types.QueryStatus
		var err error = nil
		for range 10 {
			if len(inst.queryId) == 0 {
				break
			}

			results, status, err = inst.serviceCtx.Api.GetInightsQueryResults(inst.queryId)

			switch status {
			case types.QueryStatusRunning, types.QueryStatusScheduled:
				time.Sleep(2 * time.Second)
			case types.QueryStatusComplete, types.QueryStatusCancelled:
				inst.SetQueryId("")
				break
			default:
				inst.SetQueryId("")

				if err != nil {
					inst.ErrorMessageCallback(err.Error())
				} else {
					inst.ErrorMessageCallback(fmt.Sprintf("Query failed with status %s", status))
				}
				break
			}
		}

		inst.data = results
	})

	dataLoader.AsyncUpdateView(inst.rootView, func() {
		inst.populateQueryResultsTable() // update according to query status
	})
}

func (inst *InsightsQueryResultsTable) ExecuteQuery() {
	var query, err = inst.queryView.Input.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	if len(inst.selectedLogGroups) == 0 {
		inst.ErrorMessageCallback("No log groups selected")
		return
	}

	var queryIdChan = make(chan string, 1)
	go func() {
		if len(inst.queryId) > 0 {
			var _, err = inst.serviceCtx.Api.StopInightsQuery(inst.queryId)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
			inst.SetQueryId("")
		}

		var res, err = inst.serviceCtx.Api.StartInightsQuery(
			inst.selectedLogGroups,
			query.startTime,
			query.endTime,
			query.query,
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
		queryIdChan <- res
	}()

	go func() {
		inst.SetQueryId(<-queryIdChan)
		inst.RefreshResults()
	}()
}

func (inst *InsightsQueryResultsTable) StopQuery() {
	var stopSuccess = make(chan bool, 1)
	go func() {
		if len(inst.queryId) == 0 {
			stopSuccess <- true
			return
		}
		var res, err = inst.serviceCtx.Api.StopInightsQuery(inst.queryId)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
		stopSuccess <- res
	}()

	go func() {
		var _ = <-stopSuccess
		inst.SetQueryId("")
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

func (inst *InsightsQueryResultsTable) SetQueryId(id string) {
	inst.queryId = id
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

func (inst *InsightsQueryResultsTable) SetSelectedLogGroups(groups []string) {
	inst.selectedLogGroups = groups
}
