package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"fmt"
	"log"
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

type InsightsQueryResultsTable struct {
	*core.SelectableTable[string]
	*InsightsQuerySearchView
	rootView             core.View
	table                *tview.Table
	data                 [][]types.ResultField
	queryId              string
	selectedLogGroups    []string
	headingIdxMap        map[string]int
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
	ErrorMessageCallback func(text string, a ...any)
}

func NewInsightsQueryResultsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsTable {
	var selectableTable = core.NewSelectableTable[string]("", nil, app)
	var searchView = NewInsightsQuerySearchView(selectableTable, app, logger)

	var view = &InsightsQueryResultsTable{
		SelectableTable:         selectableTable,
		InsightsQuerySearchView: searchView,
		rootView:                selectableTable.Box,
		table:                   selectableTable.GetTable(),
		data:                    nil,
		queryId:                 "",
		selectedLogGroups:       nil,
		headingIdxMap:           map[string]int{},
		logger:                  logger,
		app:                     app,
		api:                     api,
		ErrorMessageCallback:    func(text string, a ...any) {},
	}

	view.populateQueryResultsTable()
	view.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshResults()
		}
		return event
	})

	view.queryView.DoneButton.SetSelectedFunc(func() {
		view.ExecuteQuery()
	})

	view.queryView.CancelButton.SetSelectedFunc(func() {
		view.StopQuery()
	})

	return view
}

func (inst *InsightsQueryResultsTable) populateQueryResultsTable() {
	inst.table.
		Clear().
		SetFixed(1, 0)

	var tableTitle = fmt.Sprintf("Query Results (%d)", len(inst.data))
	inst.rootView.SetTitle(tableTitle)

	var headingIdx = 0
	inst.headingIdxMap = map[string]int{}
	for rowIdx, rowData := range inst.data {
		for _, resField := range rowData {
			var colIdx, ok = inst.headingIdxMap[*resField.Field]
			if !ok {
				inst.headingIdxMap[*resField.Field] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellText = aws.ToString(resField.Value)
			inst.table.SetCell(rowIdx+1, colIdx, core.NewTableCell(cellText, &cellText))
		}
	}

	for heading, colIdx := range inst.headingIdxMap {
		core.SetTableHeading(inst.table, heading, colIdx)
	}

	inst.table.SetSelectable(true, true).SetSelectedStyle(
		tcell.Style{}.Background(core.MoreContrastBackgroundColor),
	)

	inst.table.Select(1, 0)
	inst.table.ScrollToBeginning()
}

func (inst *InsightsQueryResultsTable) RefreshResults() {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var results [][]types.ResultField
		var status types.QueryStatus
		var err error = nil
		for range 10 {
			if len(inst.queryId) == 0 {
				break
			}

			results, status, err = inst.api.GetInightsQueryResults(inst.queryId)

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
	var query, err = inst.queryView.GenerateQuery()
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
			var _, err = inst.api.StopInightsQuery(inst.queryId)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
			inst.SetQueryId("")
		}

		var res, err = inst.api.StartInightsQuery(
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
		var res, err = inst.api.StopInightsQuery(inst.queryId)
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
	var ptrCol, ok = inst.headingIdxMap["@ptr"]
	if !ok {
		return ""
	}
	return inst.GetPrivateData(row, ptrCol)
}

func (inst *InsightsQueryResultsTable) SetSelectedLogGroups(groups []string) {
	inst.selectedLogGroups = groups
}
