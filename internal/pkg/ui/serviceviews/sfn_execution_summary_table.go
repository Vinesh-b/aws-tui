package serviceviews

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachineExecutionSummaryTable struct {
	*core.DetailsTable
	selectedExecutionArn string

	data   *sfn.DescribeExecutionOutput
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.StateMachineApi
}

func NewStateMachineExecutionSummaryTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionSummaryTable {

	var table = &StateMachineExecutionSummaryTable{
		DetailsTable:         core.NewDetailsTable("Execution Summary"),
		selectedExecutionArn: "",

		data:   nil,
		logger: logger,
		app:    app,
		api:    api,
	}

	table.populateTable()
	table.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			table.RefreshExecutionDetails(table.selectedExecutionArn, true)
		}
		return event
	})

	return table
}

func (inst *StateMachineExecutionSummaryTable) populateTable() {
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
	inst.Table.Select(0, 0)
	inst.Table.ScrollToBeginning()
}

func (inst *StateMachineExecutionSummaryTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.selectedExecutionArn = executionArn
	var resultChannel = make(chan struct{})

	go func() {
		inst.data = inst.api.DescribeExecution(executionArn)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateTable()
	})
}
