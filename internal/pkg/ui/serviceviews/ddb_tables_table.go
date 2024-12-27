package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBTablesTable struct {
	*core.SelectableTable[any]
	selectedTable string
	data          []string
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.DynamoDBApi
}

func NewDynamoDBTablesTable(
	app *tview.Application,
	api *awsapi.DynamoDBApi,
	logger *log.Logger,
) *DynamoDBTablesTable {

	var table = &DynamoDBTablesTable{
		SelectableTable: core.NewSelectableTable[any](
			"DynamoDB Tables",
			core.TableRow{
				"Name",
			},
		),
		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateTablesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			table.RefreshTables(true)
		}
	})

	return table
}

func (inst *DynamoDBTablesTable) populateTablesTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{row})
	}

	inst.SetData(tableData)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *DynamoDBTablesTable) RefreshTables(force bool) {
	var resultChannel = make(chan struct{})
	var search = inst.GetSearchText()

	go func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			inst.data = inst.api.ListTables(force)
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateTablesTable()
	})
}

func (inst *DynamoDBTablesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTable = inst.Table.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *DynamoDBTablesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshTables(true)
		}
		return capture(event)
	})

}

func (inst *DynamoDBTablesTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.Table.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
}

func (inst *DynamoDBTablesTable) GetSelectedTable() string {
	return inst.selectedTable
}
