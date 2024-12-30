package serviceviews

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
	*core.SearchableView
	Table          *tview.Table
	selectedLambda string
	data           [][]types.ResultField
	queryId        string
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.CloudWatchLogsApi
}

func NewInsightsQueryResultsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *InsightsQueryResultsTable {
	var table = tview.NewTable()

	var view = &InsightsQueryResultsTable{
		SearchableView: core.NewSearchableView(table),
		Table:          table,
		data:           nil,
		queryId:        "",
		logger:         logger,
		app:            app,
		api:            api,
	}

	view.HighlightSearch = true
	view.populateQueryResultsTable()
	view.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			view.RefreshResults()
		}
		return event
	})

	return view
}

func (inst *InsightsQueryResultsTable) populateQueryResultsTable() {
	inst.Table.
		Clear().
		SetBorders(false).
		SetFixed(1, 0)
	inst.Table.
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	var tableTitle = fmt.Sprintf("Query Results (%d)", len(inst.data))
	inst.Table.SetTitle(tableTitle)

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
	var resultChannel = make(chan struct{})

	go func() {
		var results [][]types.ResultField
		var status types.QueryStatus
		for range 10 {
			results, status = inst.api.GetInightsQueryResults(inst.queryId)
			if status == types.QueryStatusRunning || status == types.QueryStatusScheduled {
				time.Sleep(2 * time.Second)
			} else {
				break
			}
		}

		inst.data = results
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateQueryResultsTable() // update according to query status
	})
}

func (inst *InsightsQueryResultsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.Table.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
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
