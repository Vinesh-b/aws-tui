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

const sfnExecutionArnCol = 0

type StateMachineExecutionsTable struct {
	*core.SelectableTable[string]
	*SfnExecutionsQuerySearchView
	selectedFunctionArn  string
	selectedExecutionArn string
	data                 []types.ExecutionListItem
	filtered             []types.ExecutionListItem
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.StateMachineApi
}

func NewStateMachineExecutionsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	logger *log.Logger,
) *StateMachineExecutionsTable {
	var selectableTable = core.NewSelectableTable[string](
		"Executions",
		core.TableRow{
			"Execution Arn",
			"Status",
			"Start Date",
			"Stop Date",
		},
	)
	var searchView = NewSfnExecutionsQuerySearchView(selectableTable, app, logger)

	var table = &StateMachineExecutionsTable{
		SfnExecutionsQuerySearchView: searchView,
		SelectableTable:              selectableTable,
		selectedFunctionArn:          "",
		selectedExecutionArn:         "",
		data:                         nil,
		logger:                       logger,
		app:                          app,
		api:                          api,
	}

	var endTime = time.Now()
	var startTime = endTime.Add(-24 * 30 * 15 * time.Hour)
	table.queryView.SetDefaultTimes(startTime, endTime)

	table.populateExecutionsTable(true)
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSelectionChangedFunc(func(row, column int) {})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			var endTime = time.Now()
			var startTime = endTime.Add(-24 * 30 * 15 * time.Hour)
			table.queryView.SetDefaultTimes(startTime, endTime)

			table.RefreshExecutions(true)
		case core.APP_KEY_BINDINGS.NextPage:
			table.RefreshExecutions(false)
		}
		return event
	})

	table.queryView.DoneButton.SetSelectedFunc(func() {
		table.RefreshExecutions(true)
	})

	return table
}

func (inst *StateMachineExecutionsTable) populateExecutionsTable(force bool) {
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

	if !force {
		inst.ExtendData(tableData, privateData)
		return
	}

	inst.SetData(tableData, privateData, sfnExecutionArnCol)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *StateMachineExecutionsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		inst.selectedExecutionArn = inst.GetPrivateData(row, sfnExecutionArnCol)
		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) FilterByStatus(
	data []types.ExecutionListItem, status string,
) []types.ExecutionListItem {
	if status == "ALL" {
		return data
	}

	var result = []types.ExecutionListItem{}
	for _, exe := range data {
		if exe.Status == types.ExecutionStatus(status) {
			result = append(result, exe)
		}
	}
	return result
}

func (inst *StateMachineExecutionsTable) RefreshExecutions(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	var query, err = inst.queryView.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedFunctionArn) > 0 {
			var err error = nil

			inst.data, err = inst.api.ListExecutions(
				inst.selectedFunctionArn,
				query.startTime,
				query.endTime,
				reset,
			)

			inst.data = inst.FilterByStatus(inst.data, query.status)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateExecutionsTable(reset)
	})
}

func (inst *StateMachineExecutionsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedExecutionArn = inst.GetPrivateData(row, sfnExecutionArnCol)

		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) GetSeletedExecutionArn() string {
	return inst.selectedExecutionArn
}

func (inst *StateMachineExecutionsTable) SetSeletedFunctionArn(functionArn string) {
	inst.selectedFunctionArn = functionArn
}
