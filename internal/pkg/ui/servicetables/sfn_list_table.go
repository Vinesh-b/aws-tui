package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const sfnFunctionNameCol = 0

type StateMachinesListTable struct {
	*core.SelectableTable[types.StateMachineListItem]
	selectedFunction types.StateMachineListItem
	data             []types.StateMachineListItem
	filtered         []types.StateMachineListItem
	logger           *log.Logger
	app              *tview.Application
	api              *awsapi.StateMachineApi
}

func NewStateMachinesListTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesListTable {

	var table = &StateMachinesListTable{
		SelectableTable: core.NewSelectableTable[types.StateMachineListItem](
			"State Machines",
			core.TableRow{
				"Name",
				"Type",
				"Creation Date",
			},
			app,
		),

		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateStateMachinesTable(table.data)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var search = table.GetSearchText()
			table.FilterByName(search)
		}
	})

	return table
}

func (inst *StateMachinesListTable) populateStateMachinesTable(data []types.StateMachineListItem) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			string(row.Type),
			row.CreationDate.Format(time.DateTime),
		})
	}

	inst.SetData(tableData, data, sfnFunctionNameCol)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *StateMachinesListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(name,
			inst.data,
			func(v types.StateMachineListItem) string {
				return aws.ToString(v.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStateMachinesTable(inst.filtered)
	})
}

func (inst *StateMachinesListTable) RefreshStateMachines(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.api.ListStateMachines(reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStateMachinesTable(inst.data)
	})
}

func (inst *StateMachinesListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedFunction = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *StateMachinesListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshStateMachines(true)
		}

		return capture(event)
	})
}

func (inst *StateMachinesListTable) GetSeletedFunctionArn() string {
	return aws.ToString(inst.selectedFunction.StateMachineArn)
}

func (inst *StateMachinesListTable) GetSeletedFunctionType() string {
	return string(inst.selectedFunction.Type)
}
