package cloudformation

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/gdamore/tcell/v2"
)

type StackDetailsTable struct {
	*core.DetailsTable
	data          types.StackSummary
	selectedStack types.StackSummary
	serviceCtx    *core.ServiceContext[awsapi.CloudFormationApi]
}

func NewStackDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudFormationApi],
) *StackDetailsTable {

	var view = &StackDetailsTable{
		DetailsTable: core.NewDetailsTable("Stack Details", serviceContext.AppContext),
		data:         types.StackSummary{},
		serviceCtx:   serviceContext,
	}

	view.populateStackDetailsTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshDetails(view.data)
			return nil
		}
		return event
	})

	return view
}

func (inst *StackDetailsTable) populateStackDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.StackName)},
		{"Stack ID", aws.ToString(inst.data.StackId)},
		{"Description", aws.ToString(inst.data.TemplateDescription)},
		{"Status", string(inst.data.StackStatus)},
		{"Status Reason", aws.ToString(inst.data.StackStatusReason)},
		{"Created", core.FormatTime(inst.data.CreationTime)},
		{"Last Updated", core.FormatTime(inst.data.LastUpdatedTime)},
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
}

func (inst *StackDetailsTable) RefreshDetails(data types.StackSummary) {
	inst.data = data
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateStackDetailsTable()
	})
}
