package servicetables

import (
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/gdamore/tcell/v2"
)

type SfnExecutionSummaryTable struct {
	*core.DetailsTable
	selectedExecutionArn string

	data       *sfn.DescribeExecutionOutput
	serviceCtx *core.ServiceContext[awsapi.StateMachineApi]
}

func NewSfnExecutionSummaryTable(
	serviceContext *core.ServiceContext[awsapi.StateMachineApi],
) *SfnExecutionSummaryTable {

	var table = &SfnExecutionSummaryTable{
		DetailsTable:         core.NewDetailsTable("Execution Summary", serviceContext.AppContext),
		selectedExecutionArn: "",

		data:       nil,
		serviceCtx: serviceContext,
	}

	table.populateTable()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			table.RefreshExecutionDetails(table.selectedExecutionArn, true)
		}
		return event
	})

	return table
}

func (inst *SfnExecutionSummaryTable) populateTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = []core.TableRow{
			{"Name", aws.ToString(inst.data.Name)},
			{"Execution Arn", aws.ToString(inst.data.ExecutionArn)},
			{"StateMachine Arn", aws.ToString(inst.data.StateMachineArn)},
			{"Status", string(inst.data.Status)},
			{"Start Date", inst.data.StartDate.Format(time.DateTime)},
			{"Stop Date", inst.data.StopDate.Format(time.DateTime)},
		}
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *SfnExecutionSummaryTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.selectedExecutionArn = executionArn
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.data, err = inst.serviceCtx.Api.DescribeExecution(executionArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}
