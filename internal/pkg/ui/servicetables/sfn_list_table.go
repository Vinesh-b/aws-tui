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
	*core.SelectableTable[string]
	selectedFunctionArn string
	data                map[string]types.StateMachineListItem
	logger              *log.Logger
	app                 *tview.Application
	api                 *awsapi.StateMachineApi
}

func NewStateMachinesListTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachinesListTable {

	var table = &StateMachinesListTable{
		SelectableTable: core.NewSelectableTable[string](
			"State Machines",
			core.TableRow{
				"Name",
				"Creation Date",
			},
		),

		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateStateMachinesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			table.RefreshStateMachines(false)
		}
	})

	return table
}

func (inst *StateMachinesListTable) populateStateMachinesTable() {
	var tableData []core.TableRow
	var privateData []string
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			row.CreationDate.Format(time.DateTime),
		})
		privateData = append(privateData, aws.ToString(row.StateMachineArn))
	}

	inst.SetData(tableData)
	inst.SetPrivateData(privateData, sfnFunctionNameCol)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *StateMachinesListTable) RefreshStateMachines(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			var err error = nil
			inst.data, err = inst.api.ListStateMachines(force)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStateMachinesTable()
	})
}

func (inst *StateMachinesListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		var ref = inst.GetCell(row, 0).Reference
		if row < 1 || ref == nil {
			return
		}
		inst.selectedFunctionArn = ref.(string)

		handler(row, column)
	})
}

func (inst *StateMachinesListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshStateMachines(true)
		}

		return capture(event)
	})
}

func (inst *StateMachinesListTable) GetSeletedFunctionArn() string {
	return inst.selectedFunctionArn
}
