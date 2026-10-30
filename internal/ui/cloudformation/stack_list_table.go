package cloudformation

import (
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
)

type StackListTable struct {
	*core.SelectableTable[types.StackSummary]
	data          []types.StackSummary
	selectedStack types.StackSummary
	serviceCtx    *core.ServiceContext[awsapi.CloudFormationApi]
}

func NewStackListTable(
	serviceContext *core.ServiceContext[awsapi.CloudFormationApi],
) *StackListTable {

	var view = &StackListTable{
		SelectableTable: core.NewSelectableTable[types.StackSummary](
			"Stacks",
			core.TableRow{
				"StackName",
				"Status",
				"LastUpdated",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0),
		data:       nil,
		serviceCtx: serviceContext,
	}

	view.populateStacksTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *StackListTable) populateStacksTable(data []types.StackSummary) {
	var tableData []core.TableRow
	for _, row := range data {
		var lastUpdated = "-"
		if row.LastUpdatedTime != nil {
			lastUpdated = row.LastUpdatedTime.Format(time.DateTime)
		}
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.StackName),
			string(row.StackStatus),
			lastUpdated,
		})
	}

	inst.SetData(tableData, data, 0)
	inst.Select(1, 0)
}

func (inst *StackListTable) RefreshStacks(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListStacks(inst.serviceCtx.Ctx, reset)
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
		inst.populateStacksTable(inst.data)
	})
}

func (inst *StackListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedStack = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *StackListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshStacks(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *StackListTable) GetSelectedStackName() string {
	return aws.ToString(inst.selectedStack.StackName)
}

func (inst *StackListTable) GetSelectedStack() types.StackSummary {
	return inst.selectedStack
}
