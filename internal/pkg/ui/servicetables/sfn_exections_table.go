package servicetables

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

type ExecutionItem struct {
	*types.ExecutionListItem
	logGroup         *string
	StateMachineType string
}

const sfnExecutionArnCol = 0

type StateMachineExecutionsTable struct {
	*core.SelectableTable[ExecutionItem]
	queryView           *FloatingSfnExecutionsQueryInputView
	selectedFunctionArn string
	data                []ExecutionItem
	filtered            []ExecutionItem
	selectedExecution   ExecutionItem
	logger              *log.Logger
	app                 *tview.Application
	api                 *awsapi.StateMachineApi
	cwlApi              *awsapi.CloudWatchLogsApi
}

func NewStateMachineExecutionsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	cwlApi *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *StateMachineExecutionsTable {
	var selectableTable = core.NewSelectableTable[ExecutionItem](
		"Executions",
		core.TableRow{
			"Execution Id",
			"Status",
			"Start Date",
			"Stop Date",
		},
		app,
	)

	var searchView = NewFloatingSfnExecutionsQueryInputView(app, logger)
	selectableTable.AddKeyToggleOverlay("QUERY", searchView, core.APP_KEY_BINDINGS.TableQuery)

	var table = &StateMachineExecutionsTable{
		queryView:           searchView,
		SelectableTable:     selectableTable,
		selectedFunctionArn: "",
		selectedExecution:   ExecutionItem{ExecutionListItem: &types.ExecutionListItem{}},
		data:                nil,
		logger:              logger,
		app:                 app,
		api:                 api,
		cwlApi:              cwlApi,
	}

	var endTime = time.Now()
	var startTime = endTime.Add(-24 * 1 * time.Hour)
	table.queryView.Input.SetDefaultTimes(startTime, endTime)

	table.HighlightSearch = true
	table.populateExecutionsTable(true)
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSelectionChangedFunc(func(row, column int) {})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			var endTime = time.Now()
			var startTime = endTime.Add(-24 * 1 * time.Hour)
			table.queryView.Input.SetDefaultTimes(startTime, endTime)

			if table.selectedExecution.StateMachineType == "STANDARD" {
				table.RefreshExecutions(true)
			} else {
				table.RefreshExpressExecutions(aws.ToString(table.selectedExecution.logGroup), true)
			}
		case core.APP_KEY_BINDINGS.LoadMoreData:
			table.RefreshExecutions(false)
		}
		return event
	})

	table.queryView.Input.DoneButton.SetSelectedFunc(func() {
		if table.selectedExecution.StateMachineType == "STANDARD" {
			table.RefreshExecutions(true)
		} else {
			table.RefreshExpressExecutions(aws.ToString(table.selectedExecution.logGroup), true)
		}
	})

	return table
}

func (inst *StateMachineExecutionsTable) populateExecutionsTable(force bool) {
	var tableData []core.TableRow

	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			string(row.Status),
			aws.ToTime(row.StartDate).Format(time.DateTime),
			aws.ToTime(row.StopDate).Format(time.DateTime),
		})
	}

	if !force {
		inst.ExtendData(tableData, inst.data)
		return
	}

	inst.SetData(tableData, inst.data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *StateMachineExecutionsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		inst.selectedExecution = inst.GetPrivateData(row, sfnExecutionArnCol)
		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) FilterByExecutionId(
	data []ExecutionItem, executionArn string,
) []ExecutionItem {
	if len(executionArn) == 0 {
		return data
	}

	var result = []ExecutionItem{}
	for _, exe := range data {
		if aws.ToString(exe.Name) == executionArn {
			result = append(result, exe)
			break
		}
	}
	return result
}

func (inst *StateMachineExecutionsTable) FilterByStatus(
	data []ExecutionItem, status string,
) []ExecutionItem {
	if status == "ALL" {
		return data
	}

	var result = []ExecutionItem{}
	for _, exe := range data {
		if exe.Status == types.ExecutionStatus(status) {
			result = append(result, exe)
		}
	}
	return result
}

func (inst *StateMachineExecutionsTable) RefreshExpressExecutions(logGroup string, reset bool) {
	inst.selectedExecution.logGroup = &logGroup
	var query, err = inst.queryView.Input.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	var insightsQuery = InsightsQuery{
		query:     `fields @message | filter type=~"Execution"`,
		startTime: query.startTime,
		endTime:   query.endTime,
	}

	var insightsQueryRunner = NewInsightsQueryRunner(inst.app, inst.cwlApi)
	insightsQueryRunner.ErrorMessageCallback = inst.ErrorMessageCallback

	var resultsChan = make(chan [][]cwlTypes.ResultField)

	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	dataLoader.AsyncLoadData(func() {
		insightsQueryRunner.ExecuteInsightsQuery(insightsQuery, []string{logGroup}, resultsChan)
		var insightsResults = <-resultsChan
		if len(insightsResults) == 0 {
			return
		}

		var executions = map[string]*ExecutionItem{}

		var tableData = []ExecutionItem{}
		for _, message := range insightsResults {
			var stateMachineStep StateMachineStep
			for _, resultField := range message {
				var field = aws.ToString(resultField.Field)
				switch field {
				case "@message":
					if err := json.Unmarshal([]byte(aws.ToString(resultField.Value)), &stateMachineStep); err != nil {
						inst.ErrorMessageCallback("Failed to parse state: %s", err.Error())
					}
				}
			}
			var splitArn = strings.SplitN(stateMachineStep.ExecutionArn, ":", 8)
			var name = splitArn[7]
			var currentExe = &ExecutionItem{
				ExecutionListItem: &types.ExecutionListItem{},
				StateMachineType:  "EXPRESS",
			}
			if execution, ok := executions[stateMachineStep.ExecutionArn]; !ok {
				executions[stateMachineStep.ExecutionArn] = currentExe
			} else {
				currentExe = execution
			}

			var timestamp = stateMachineStep.Timestamp

			switch stateMachineStep.StateType {
			case types.HistoryEventTypeExecutionStarted:
				currentExe.StartDate = aws.Time(time.UnixMilli(timestamp))

			case
				types.HistoryEventTypeExecutionFailed,
				types.HistoryEventTypeExecutionSucceeded,
				types.HistoryEventTypeExecutionTimedOut,
				types.HistoryEventTypeExecutionAborted:

				switch stateMachineStep.StateType {
				case types.HistoryEventTypeExecutionFailed:
					currentExe.Status = types.ExecutionStatusFailed
				case types.HistoryEventTypeExecutionSucceeded:
					currentExe.Status = types.ExecutionStatusSucceeded
				case types.HistoryEventTypeExecutionTimedOut:
					currentExe.Status = types.ExecutionStatusTimedOut
				case types.HistoryEventTypeExecutionAborted:
					currentExe.Status = types.ExecutionStatusAborted
				}
				currentExe.StopDate = aws.Time(time.UnixMilli(timestamp))
			}

			currentExe.ExecutionArn = aws.String(stateMachineStep.ExecutionArn)
			currentExe.Name = aws.String(name)
			currentExe.StateMachineArn = nil
			currentExe.logGroup = aws.String(logGroup)
		}

		for _, exe := range executions {
			// Only collect full execution data
			if exe.StartDate != nil && exe.StopDate != nil {
				tableData = append(tableData, *exe)
			}
		}
		sort.Slice(tableData, func(i, j int) bool {
			var start1 = aws.ToTime(tableData[i].StartDate)
			var start2 = aws.ToTime(tableData[j].StartDate)
			return start1.After(start2)
		})

		inst.data = tableData
	})

	dataLoader.AsyncUpdateView(inst.SelectableTable.Box, func() {
		inst.populateExecutionsTable(reset)
	})
}

func (inst *StateMachineExecutionsTable) RefreshExecutions(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	var query, err = inst.queryView.Input.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	dataLoader.AsyncLoadData(func() {
		if len(inst.selectedFunctionArn) > 0 {
			var data, err = inst.api.ListExecutions(
				inst.selectedFunctionArn,
				query.startTime,
				query.endTime,
				reset,
			)

			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}

			inst.data = nil
			for _, d := range data {
				inst.data = append(inst.data, ExecutionItem{
					ExecutionListItem: &d,
					logGroup:          nil,
					StateMachineType:  "STANDARD",
				})
			}

			inst.data = inst.FilterByStatus(inst.data, query.status)
			inst.data = inst.FilterByExecutionId(inst.data, query.executionArn)
		}
	})

	dataLoader.AsyncUpdateView(inst.SelectableTable.Box, func() {
		inst.populateExecutionsTable(reset)
	})
}

func (inst *StateMachineExecutionsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedExecution = inst.GetPrivateData(row, sfnExecutionArnCol)

		handler(row, column)
	})
}

func (inst *StateMachineExecutionsTable) GetSeletedExecution() ExecutionItem {
	return inst.selectedExecution
}

func (inst *StateMachineExecutionsTable) GetSeletedExecutionArn() string {
	return aws.ToString(inst.selectedExecution.ExecutionArn)
}

func (inst *StateMachineExecutionsTable) SetSeletedFunctionArn(functionArn string) {
	inst.selectedFunctionArn = functionArn
}
