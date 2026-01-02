package servicetables

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/gdamore/tcell/v2"
)

type ExecutionItem struct {
	*types.ExecutionListItem
	logGroup         *string
	StateMachineType string
}

const sfnExecutionArnCol = 0

type SfnExecutionsTable struct {
	*core.SelectableTable[ExecutionItem]
	queryView         *FloatingSfnExecutionsQueryInputView
	selectedFunction  types.StateMachineListItem
	data              []ExecutionItem
	filtered          []ExecutionItem
	selectedExecution ExecutionItem
	appCtx            *core.AppContext
	api               *awsapi.StateMachineApi
	cwlApi            *awsapi.CloudWatchLogsApi
}

func NewSfnExecutionsTable(
	appCtx *core.AppContext,
	api *awsapi.StateMachineApi,
	cwlApi *awsapi.CloudWatchLogsApi,
) *SfnExecutionsTable {
	var selectableTable = core.NewSelectableTable[ExecutionItem](
		"Executions",
		core.TableRow{
			"Execution Id",
			"Status",
			"Start Date",
			"Stop Date",
			"Duration",
		},
		appCtx,
	)

	var searchView = NewFloatingSfnExecutionsQueryInputView(appCtx)
	selectableTable.AddRuneToggleOverlay("QUERY", searchView, core.APP_KEY_BINDINGS.TableQuery, false)

	var table = &SfnExecutionsTable{
		queryView:         searchView,
		SelectableTable:   selectableTable,
		selectedFunction:  types.StateMachineListItem{},
		selectedExecution: ExecutionItem{ExecutionListItem: &types.ExecutionListItem{}},
		data:              nil,
		appCtx:            appCtx,
		api:               api,
		cwlApi:            cwlApi,
	}

	var endTime = time.Now()
	var startTime = endTime.Add(-24 * 1 * time.Hour)
	table.queryView.Input.SetDefaultTimes(startTime, endTime)

	table.HighlightSearch = true
	table.populateTable(true)
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSelectionChangedFunc(func(row, column int) {})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			var endTime = time.Now()
			var startTime = endTime.Add(-24 * 1 * time.Hour)
			table.queryView.Input.SetDefaultTimes(startTime, endTime)

			switch table.selectedFunction.Type {
			case types.StateMachineTypeStandard:
				table.RefreshExecutions(true)
			case types.StateMachineTypeExpress:
				table.RefreshExpressExecutions(aws.ToString(table.selectedExecution.logGroup), true)
			default:
				table.ErrorMessageCallback(
					"Unsupported type: %s", table.selectedExecution.StateMachineType,
				)
			}
		case core.APP_KEY_BINDINGS.LoadMoreData:
			table.RefreshExecutions(false)
		}
		return event
	})

	table.queryView.Input.DoneButton.SetSelectedFunc(func() {
		switch table.selectedFunction.Type {
		case types.StateMachineTypeStandard:
			table.RefreshExecutions(true)
		case types.StateMachineTypeExpress:
			table.RefreshExpressExecutions(aws.ToString(table.selectedExecution.logGroup), true)
		default:
			table.ErrorMessageCallback(
				"Unsupported type: %s", table.selectedExecution.StateMachineType,
			)
		}
	})

	table.HelpView.View.
		AddItem("f", "Jump to next search result", nil).
		AddItem("F", "Jump to previous search result", nil).
		AddItem("q", "To show query view", nil)

	return table
}

func (inst *SfnExecutionsTable) populateTable(force bool) {
	var tableData []core.TableRow

	for _, row := range inst.data {
		var startTime = aws.ToTime(row.StartDate)
		var endTime = aws.ToTime(row.StopDate)
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Name),
			string(row.Status),
			startTime.Format(time.DateTime),
			endTime.Format(time.DateTime),
			endTime.Sub(startTime).String(),
		})
	}

	if !force {
		inst.ExtendData(tableData, inst.data)
		return
	}

	inst.SetTitleExtra(aws.ToString(inst.selectedFunction.Name))
	inst.SetData(tableData, inst.data, 0)
	inst.GetCell(0, 0).SetExpansion(1)

	// temp hack to colour successful and failed executions
	var table = inst.GetTable()
	var rows = table.GetRowCount()
	for r := range rows {
		var cell = table.GetCell(r, 1)
		if cell.Text == "SUCCEEDED" {
			cell.SetStyle(tcell.Style{}.Foreground(tcell.ColorForestGreen))
		} else if cell.Text == "FAILED" {
			cell.SetStyle(tcell.Style{}.Foreground(tcell.ColorIndianRed))
		}
	}
}

func (inst *SfnExecutionsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		inst.selectedExecution = inst.GetPrivateData(row, sfnExecutionArnCol)
		handler(row, column)
	})
}

func (inst *SfnExecutionsTable) FilterByExecutionId(
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

func (inst *SfnExecutionsTable) FilterByStatus(
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

func (inst *SfnExecutionsTable) RefreshExpressExecutions(logGroup string, reset bool) {
	inst.selectedExecution.logGroup = &logGroup
	var query, err = inst.queryView.Input.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	var executionFilter = "| filter execution_arn like /" + query.executionArn + "/"

	var insightsQuery = InsightsQuery{
		query:     `fields @message | filter type=~"Execution"` + executionFilter,
		startTime: query.startTime,
		endTime:   query.endTime,
	}

	var insightsQueryRunner = NewInsightsQueryRunner(inst.appCtx.App, inst.cwlApi)
	insightsQueryRunner.ErrorMessageCallback = inst.ErrorMessageCallback

	var resultsChan = make(chan [][]cwlTypes.ResultField)

	var dataLoader = core.NewUiDataLoader(inst.appCtx.App, 30)
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
		inst.populateTable(reset)
	})
}

func (inst *SfnExecutionsTable) RefreshExecutions(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.appCtx.App, 10)
	var query, err = inst.queryView.Input.GenerateQuery()
	if err != nil {
		inst.ErrorMessageCallback(err.Error())
		return
	}

	dataLoader.AsyncLoadData(func() {
		var selectedFunctionArn = aws.ToString(inst.selectedFunction.StateMachineArn)
		if len(selectedFunctionArn) > 0 {
			var data, err = inst.api.ListExecutions(
				selectedFunctionArn,
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
		inst.populateTable(reset)
	})
}

func (inst *SfnExecutionsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedExecution = inst.GetPrivateData(row, sfnExecutionArnCol)

		handler(row, column)
	})
}

func (inst *SfnExecutionsTable) GetSeletedExecution() ExecutionItem {
	return inst.selectedExecution
}

func (inst *SfnExecutionsTable) GetSeletedExecutionArn() string {
	return aws.ToString(inst.selectedExecution.ExecutionArn)
}

func (inst *SfnExecutionsTable) SetSeletedFunction(function types.StateMachineListItem) {
	inst.selectedFunction = function
}
