package dynamodb

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/gdamore/tcell/v2"
)

type DynamoDBTablesTable struct {
	*core.SelectableTable[any]
	data          []string
	allTables     []string
	selectedTable string
	serviceCtx    *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDBTablesTable(
	serviceContext *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBTablesTable {

	var table = &DynamoDBTablesTable{
		SelectableTable: core.NewSelectableTable[any](
			"DynamoDB Tables",
			core.TableRow{
				"Name",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1),
		data:       nil,
		allTables:  nil,
		serviceCtx: serviceContext,
	}

	table.populateTablesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
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

	inst.SetData(tableData, nil, 0)
	inst.Select(1, 0)
}

func (inst *DynamoDBTablesTable) RefreshTables(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = utils.FuzzySearch(search, inst.allTables, func(t string) string {
				return t
			})
		} else {
			var err error = nil
			inst.allTables, err = inst.serviceCtx.Api.ListTables(inst.serviceCtx.Ctx, force)
			inst.data = inst.allTables
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTablesTable()
	})
}

func (inst *DynamoDBTablesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedTable = inst.GetCell(row, 0).Text
		handler(row, column)
	})
}

func (inst *DynamoDBTablesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshTables(true)
			return nil
		}
		return capture(event)
	})

}

func (inst *DynamoDBTablesTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
}

func (inst *DynamoDBTablesTable) GetSelectedTable() string {
	return inst.selectedTable
}
