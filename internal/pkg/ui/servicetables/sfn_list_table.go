package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
)

const sfnFunctionNameCol = 0

type SfnListTable struct {
	*core.SelectableTable[types.StateMachineListItem]
	selectedFunction types.StateMachineListItem
	data             []types.StateMachineListItem
	filtered         []types.StateMachineListItem
	logger           *log.Logger
	serviceCtx       *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnListTable(
	serviceViewCtx *core.ServiceContext[awsapi.StateMachineApi],
) *SfnListTable {

	var table = &SfnListTable{
		SelectableTable: core.NewSelectableTable[types.StateMachineListItem](
			"State Machines",
			core.TableRow{
				"Name",
				"Type",
				"Creation Date",
			},
			serviceViewCtx.AppContext,
		),

		data:       nil,
		serviceCtx: serviceViewCtx,
	}

	table.populateTable(table.data)
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

func (inst *SfnListTable) populateTable(data []types.StateMachineListItem) {
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

func (inst *SfnListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(name,
			inst.data,
			func(v types.StateMachineListItem) string {
				return aws.ToString(v.Name)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable(inst.filtered)
	})
}

func (inst *SfnListTable) RefreshStateMachines(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListStateMachines(reset)
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
		inst.populateTable(inst.data)
	})
}

func (inst *SfnListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedFunction = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SfnListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshStateMachines(true)
		}

		return capture(event)
	})
}

func (inst *SfnListTable) GetSeletedFunction() types.StateMachineListItem {
	return inst.selectedFunction
}

func (inst *SfnListTable) GetSeletedFunctionName() string {
	return aws.ToString(inst.selectedFunction.Name)
}

func (inst *SfnListTable) GetSeletedFunctionArn() string {
	return aws.ToString(inst.selectedFunction.StateMachineArn)
}

func (inst *SfnListTable) GetSeletedFunctionType() types.StateMachineType {
	return inst.selectedFunction.Type
}
