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

type InsightsQueryResultsTable struct {
	*InsightsQuerySearchView
	Table                *tview.Table
	data                 [][]types.ResultField
	queryId              string
	selectedLogGroups    []string
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.CloudWatchLogsApi
	ErrorMessageCallback func(text string)
}

func NewInsightsQueryResultsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsTable {
	var table = tview.NewTable()

	var view = &InsightsQueryResultsTable{
		InsightsQuerySearchView: NewInsightsQuerySearchView(table, app, logger),
		Table:                   table,
		data:                    nil,
		queryId:                 "",
		logger:                  logger,
		app:                     app,
		api:                     api,
		ErrorMessageCallback:    func(text string) {},
	}

	view.populateQueryResultsTable()
	view.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
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
	inst.Table.
		Clear().
		SetBorders(false).
		SetFixed(1, 0)
	inst.
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var tableTitle = fmt.Sprintf("Query Results (%d)", len(inst.data))
	inst.SetTitle(tableTitle)

	var headingIdx = 0
	var headingIdxMap = make(map[string]int)
	for rowIdx, rowData := range inst.data {
		for _, resField := range rowData {
			var colIdx, ok = headingIdxMap[*resField.Field]
			if !ok {
				headingIdxMap[*resField.Field] = headingIdx
				colIdx = headingIdx
				headingIdx++
			}

			var cellData = fmt.Sprintf("%s", aws.ToString(resField.Value))
			var previewText = core.ClampStringLen(&cellData, 100)
			inst.Table.SetCell(rowIdx+1, colIdx, tview.NewTableCell(previewText).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}

	for heading, colIdx := range headingIdxMap {
		inst.Table.SetCell(0, colIdx, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}

	if len(inst.data) > 0 {
		inst.Table.SetSelectable(true, true).SetSelectedStyle(
			tcell.Style{}.Background(core.MoreContrastBackgroundColor),
		)
	}
	inst.Table.Select(1, 0)
	inst.Table.ScrollToBeginning()
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

	dataLoader.AsyncUpdateView(inst.Box, func() {
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
	inst.Table.SetSelectedFunc(func(row, column int) {
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
	inst.Table.SetSelectionChangedFunc(handler)
	return inst
}

func (inst *InsightsQueryResultsTable) SetQueryId(id string) {
	inst.queryId = id
}

func (inst *InsightsQueryResultsTable) GetColumnCount() int {
	return inst.Table.GetColumnCount()
}

func (inst *InsightsQueryResultsTable) GetRecordPtr(row int) string {
	var lastCol = inst.Table.GetColumnCount() - 1
	var msg = inst.Table.GetCell(row, lastCol).Reference
	if row < 1 || msg == nil {
		return ""
	}
	return msg.(string)
}

func (inst *InsightsQueryResultsTable) GetPrivateData(row int, column int) string {
	var ref = inst.Table.GetCell(row, column).Reference
	if ref == nil {
		return ""
	}

	return ref.(string)
}

func (inst *InsightsQueryResultsTable) SetSelectedLogGroups(groups []string) {
	inst.selectedLogGroups = groups
}
