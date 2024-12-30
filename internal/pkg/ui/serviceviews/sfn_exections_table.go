package serviceviews

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/rivo/tview"
)

const sfnExecutionArnCol = 0

type StateMachineExecutionsTable struct {
	*core.SelectableTable[string]
	nextToken            *string
	selectedFunctionArn  string
	currentSearch        string
	selectedExecutionArn string
	data                 []types.ExecutionListItem
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.StateMachineApi
}

func NewStateMachineExecutionsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionsTable {

	var table = &StateMachineExecutionsTable{
		SelectableTable: core.NewSelectableTable[string](
			"Executions",
			core.TableRow{
				"Execution Arn",
				"Status",
				"Start Date",
				"Stop Date",
			},
		),
		selectedFunctionArn:  "",
		currentSearch:        "",
		selectedExecutionArn: "",
		data:                 nil,
		logger:               logger,
		app:                  app,
		api:                  api,
	}

	table.SetSelectedFunc(func(row, column int) {})
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.populateExecutionsTable()

	return table
}

func (inst *StateMachineExecutionsTable) populateExecutionsTable() {
	var tableData []core.TableRow
	var privateData []string

	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			string(row.Status),
			row.StartDate.Format(time.DateTime),
			row.StopDate.Format(time.DateTime),
		})
		privateData = append(privateData, aws.ToString(row.ExecutionArn))
	}

	inst.SetData(tableData)
	inst.SetPrivateData(privateData, sfnExecutionArnCol)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

func (inst *StateMachineExecutionsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) RefreshExecutions(force bool) {
	var resultChannel = make(chan struct{})

	go func() {
		if len(inst.selectedFunctionArn) > 0 {
			inst.data, inst.nextToken = inst.api.ListExecutions(inst.selectedFunctionArn, nil)
		}

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateExecutionsTable()
	})
}

func (inst *StateMachineExecutionsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		var ref = inst.GetCell(row, sfnExecutionArnCol).Reference
		if row < 1 || ref == nil {
			return
		}
		inst.selectedExecutionArn = ref.(string)

		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) GetSeletedExecutionArn() string {
	return inst.selectedExecutionArn
}

func (inst *StateMachineExecutionsTable) SetSeletedFunctionArn(functionArn string) {
	inst.selectedFunctionArn = functionArn
}
