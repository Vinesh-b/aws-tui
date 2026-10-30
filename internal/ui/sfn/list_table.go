package sfn

import (
	"log"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
)

const sfnFunctionNameCol = 0

type SfnListTable struct {
	*core.SelectableTable[types.StateMachineListItem]
	selectedFunction types.StateMachineListItem
	data             []types.StateMachineListItem
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
		).SetColumnExpansion(0, 1).SetFilterColumns(0),

		data:       nil,
		serviceCtx: serviceViewCtx,
	}

	table.populateTable(table.data)
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
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
	inst.Select(1, 0)
}

func (inst *SfnListTable) RefreshStateMachines(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListStateMachines(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
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
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshStateMachines(true)
			return nil
		}

		return capture(event)
	})
}

func (inst *SfnListTable) GetSelectedFunction() types.StateMachineListItem {
	return inst.selectedFunction
}

func (inst *SfnListTable) GetSelectedFunctionName() string {
	return aws.ToString(inst.selectedFunction.Name)
}

func (inst *SfnListTable) GetSelectedFunctionArn() string {
	return aws.ToString(inst.selectedFunction.StateMachineArn)
}

func (inst *SfnListTable) GetSelectedFunctionType() types.StateMachineType {
	return inst.selectedFunction.Type
}
