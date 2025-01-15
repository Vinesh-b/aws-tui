package servicetables

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StateMachineStep struct {
	Id           int64                  `json:"id,string"`
	PreviousId   int64                  `json:"previous_event_id,string"`
	Timestamp    int64                  `json:"event_timestamp,string"`
	StateType    types.HistoryEventType `json:"type"`
	RedriveCount string                 `json:"redrive_count"`
	ExecutionArn string                 `json:"execution_arn"`
	Details      struct {
		Input        string `json:"input"`
		Output       string `json:"output"`
		Name         string `json:"name"`
		Parameters   string `json:"parameters"`
		Resource     string `json:"resource"`
		ResourceType string `json:"resourceType"`
		ErrorCode    string `json:"error"`
		ErrorCause   string `json:"cause"`
	} `json:"details"`
}

type StateMachineExecutionDetailsTable struct {
	*core.SelectableTable[StateDetails]
	ExecutionHistory     *sfn.GetExecutionHistoryOutput
	selectedExecutionArn string
	selectedState        StateDetails
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.StateMachineApi
	cwlApi               *awsapi.CloudWatchLogsApi
}

func NewStateMachineExecutionDetailsTable(
	app *tview.Application,
	api *awsapi.StateMachineApi,
	cwlApi *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *StateMachineExecutionDetailsTable {

	var view = &StateMachineExecutionDetailsTable{
		SelectableTable: core.NewSelectableTable[StateDetails](
			"Execution Details",
			core.TableRow{
				"Name",
				"Type",
				"Resource",
				"Action",
				"Duration",
				"Errors",
				"Casue",
			},
			app,
		),
		ExecutionHistory:     nil,
		selectedExecutionArn: "",
		selectedState:        StateDetails{},

		logger: logger,
		app:    app,
		api:    api,
		cwlApi: cwlApi,
	}

	view.populateTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshExecutionDetails(view.selectedExecutionArn, true)
		}
		return event
	})

	return view
}

type StateDetails struct {
	Id           int64
	Name         string
	Type         string
	Input        string
	Output       string
	StartTime    time.Time
	EndTime      time.Time
	Errors       string
	Casue        string
	Resource     string
	ResourceType string
}

func (inst *StateMachineExecutionDetailsTable) parseExecutionHistory() []StateDetails {
	var results []StateDetails
	if inst.ExecutionHistory == nil {
		return results
	}

	var executionStartTime time.Time
	var taskStartTime time.Time
	for _, row := range inst.ExecutionHistory.Events {
		var stateDetails = StateDetails{
			Id:        row.Id,
			Type:      string(row.Type),
			StartTime: aws.ToTime(row.Timestamp),
			EndTime:   aws.ToTime(row.Timestamp),
		}
		switch row.Type {
		case types.HistoryEventTypeExecutionStarted:
			stateDetails.Input = aws.ToString(row.ExecutionStartedEventDetails.Input)
			executionStartTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeExecutionSucceeded:
			stateDetails.Output = aws.ToString(row.ExecutionSucceededEventDetails.Output)
			stateDetails.StartTime = executionStartTime
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeExecutionFailed:
			stateDetails.Errors = aws.ToString(row.ExecutionFailedEventDetails.Error)
			stateDetails.Casue = aws.ToString(row.ExecutionFailedEventDetails.Cause)
			stateDetails.StartTime = executionStartTime
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeExecutionAborted:
			stateDetails.Errors = aws.ToString(row.ExecutionAbortedEventDetails.Error)
			stateDetails.Casue = aws.ToString(row.ExecutionAbortedEventDetails.Cause)
			stateDetails.StartTime = executionStartTime
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeExecutionTimedOut:
			stateDetails.Errors = aws.ToString(row.ExecutionTimedOutEventDetails.Error)
			stateDetails.Casue = aws.ToString(row.ExecutionTimedOutEventDetails.Cause)
			stateDetails.StartTime = executionStartTime
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeTaskStartFailed:
			stateDetails.Errors = aws.ToString(row.TaskStartFailedEventDetails.Error)
			stateDetails.Casue = aws.ToString(row.TaskStartFailedEventDetails.Cause)
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeTaskFailed:
			stateDetails.Errors = aws.ToString(row.TaskFailedEventDetails.Error)
			stateDetails.Casue = aws.ToString(row.TaskFailedEventDetails.Cause)
			stateDetails.EndTime = aws.ToTime(row.Timestamp)

		case types.HistoryEventTypeTaskScheduled:
			stateDetails.Resource = aws.ToString(row.TaskScheduledEventDetails.Resource)
			stateDetails.ResourceType = aws.ToString(row.TaskScheduledEventDetails.ResourceType)
			stateDetails.Input = aws.ToString(row.TaskScheduledEventDetails.Parameters)

		case types.HistoryEventTypeTaskSubmitted:
			stateDetails.Output = aws.ToString(row.TaskSubmittedEventDetails.Output)
			stateDetails.Resource = aws.ToString(row.TaskSubmittedEventDetails.Resource)
			stateDetails.ResourceType = aws.ToString(row.TaskSubmittedEventDetails.ResourceType)

		case types.HistoryEventTypeTaskStarted:
			stateDetails.Resource = aws.ToString(row.TaskStartedEventDetails.Resource)
			stateDetails.ResourceType = aws.ToString(row.TaskStartedEventDetails.ResourceType)

		case types.HistoryEventTypeTaskSucceeded:
			stateDetails.Output = aws.ToString(row.TaskSucceededEventDetails.Output)
			stateDetails.Resource = aws.ToString(row.TaskSucceededEventDetails.Resource)
			stateDetails.ResourceType = aws.ToString(row.TaskSucceededEventDetails.ResourceType)

		case
			types.HistoryEventTypeTaskStateEntered,
			types.HistoryEventTypePassStateEntered,
			types.HistoryEventTypeParallelStateEntered,
			types.HistoryEventTypeMapStateEntered,
			types.HistoryEventTypeChoiceStateEntered,
			types.HistoryEventTypeSucceedStateEntered,
			types.HistoryEventTypeFailStateEntered:

			stateDetails.Name = aws.ToString(row.StateEnteredEventDetails.Name)
			stateDetails.Input = aws.ToString(row.StateEnteredEventDetails.Input)
			taskStartTime = aws.ToTime(row.Timestamp)

		case
			types.HistoryEventTypeTaskStateExited,
			types.HistoryEventTypePassStateExited,
			types.HistoryEventTypeParallelStateExited,
			types.HistoryEventTypeMapStateExited,
			types.HistoryEventTypeChoiceStateExited,
			types.HistoryEventTypeSucceedStateExited:

			var idx = slices.IndexFunc(results, func(d StateDetails) bool {
				return d.Name == aws.ToString(row.StateExitedEventDetails.Name)
			})

			if idx > -1 {
				stateDetails.Output = aws.ToString(row.StateExitedEventDetails.Output)
				stateDetails.StartTime = taskStartTime
				stateDetails.EndTime = aws.ToTime(row.Timestamp)
			}
		}

		results = append(results, stateDetails)
	}

	return results
}

func (inst *StateMachineExecutionDetailsTable) populateTable() {
	var results = inst.parseExecutionHistory()
	var tableData []core.TableRow

	for _, row := range results {
		tableData = append(tableData, core.TableRow{
			row.Name,
			row.Type,
			row.ResourceType,
			row.Resource,
			row.EndTime.Sub(row.StartTime).String(),
			row.Errors,
			row.Casue,
		})
	}

	inst.SetData(tableData, results, 0)
}

func (inst *StateMachineExecutionDetailsTable) RefreshExecutionDetails(executionArn string, force bool) {
	inst.selectedExecutionArn = executionArn
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.ExecutionHistory, err = inst.api.GetExecutionHistory(executionArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}

func (inst *StateMachineExecutionDetailsTable) RefreshExpressExecutionDetails(executionItem ExecutionItem, force bool) {
	inst.selectedExecutionArn = aws.ToString(executionItem.ExecutionArn)
	var findExecutionDetailsQuery = fmt.Sprintf(
		`fields @message | filter execution_arn="%s" | sort id asc | limit 1000`,
		inst.selectedExecutionArn,
	)

	var insightsQuery = InsightsQuery{
		query:     findExecutionDetailsQuery,
		startTime: *executionItem.StartDate,
		endTime:   *executionItem.StopDate,
	}

	var insightsQueryRunner = NewInsightsQueryRunner(inst.app, inst.cwlApi)
	insightsQueryRunner.ErrorMessageCallback = inst.ErrorMessageCallback

	var resultsChan = make(chan [][]cwlTypes.ResultField)

	var dataLoader = core.NewUiDataLoader(inst.app, 10)
	dataLoader.AsyncLoadData(func() {
		insightsQueryRunner.ExecuteInsightsQuery(insightsQuery, []string{aws.ToString(executionItem.logGroup)}, resultsChan)
		var insightsResults = <-resultsChan
		if len(insightsResults) == 0 {
			return
		}

		var historyEvents = []types.HistoryEvent{}
		for _, message := range insightsResults {
			var stateMachineStep StateMachineStep

			for _, col := range message {
				switch aws.ToString(col.Field) {
				case "@message":
					if err := json.Unmarshal([]byte(aws.ToString(col.Value)), &stateMachineStep); err != nil {
						inst.ErrorMessageCallback("Failed to parse state: %s", err.Error())
					}
				}
			}
			var id = stateMachineStep.Id
			var prevId = stateMachineStep.PreviousId
			var timestamp = stateMachineStep.Timestamp

			var executionItem = types.HistoryEvent{
				Id:              id,
				PreviousEventId: prevId,
				Timestamp:       aws.Time(time.UnixMilli(timestamp)),
				Type:            stateMachineStep.StateType,
			}

			switch stateMachineStep.StateType {
			case types.HistoryEventTypeExecutionStarted:
				executionItem.ExecutionStartedEventDetails = &types.ExecutionStartedEventDetails{
					Input: &stateMachineStep.Details.Input,
				}

			case types.HistoryEventTypeExecutionSucceeded:
				executionItem.ExecutionSucceededEventDetails = &types.ExecutionSucceededEventDetails{
					Output: &stateMachineStep.Details.Output,
				}

			case types.HistoryEventTypeExecutionFailed:
				executionItem.ExecutionFailedEventDetails = &types.ExecutionFailedEventDetails{
					Error: &stateMachineStep.Details.ErrorCode,
					Cause: &stateMachineStep.Details.ErrorCause,
				}

			case types.HistoryEventTypeExecutionAborted:
				executionItem.ExecutionAbortedEventDetails = &types.ExecutionAbortedEventDetails{
					Error: &stateMachineStep.Details.ErrorCode,
					Cause: &stateMachineStep.Details.ErrorCause,
				}

			case types.HistoryEventTypeExecutionTimedOut:
				executionItem.ExecutionTimedOutEventDetails = &types.ExecutionTimedOutEventDetails{
					Error: &stateMachineStep.Details.ErrorCode,
					Cause: &stateMachineStep.Details.ErrorCause,
				}

			case types.HistoryEventTypeTaskStartFailed:
				executionItem.TaskStartFailedEventDetails = &types.TaskStartFailedEventDetails{
					Error: &stateMachineStep.Details.ErrorCode,
					Cause: &stateMachineStep.Details.ErrorCause,
				}

			case types.HistoryEventTypeTaskFailed:
				executionItem.TaskFailedEventDetails = &types.TaskFailedEventDetails{
					Error: &stateMachineStep.Details.ErrorCode,
					Cause: &stateMachineStep.Details.ErrorCause,
				}

			case types.HistoryEventTypeTaskScheduled:
				executionItem.TaskScheduledEventDetails = &types.TaskScheduledEventDetails{
					Resource:     &stateMachineStep.Details.Resource,
					ResourceType: &stateMachineStep.Details.ResourceType,
					Parameters:   &stateMachineStep.Details.Parameters,
				}

			case types.HistoryEventTypeTaskSubmitted:
				executionItem.TaskSubmittedEventDetails = &types.TaskSubmittedEventDetails{
					Resource:     &stateMachineStep.Details.Resource,
					ResourceType: &stateMachineStep.Details.ResourceType,
					Output:       &stateMachineStep.Details.Output,
				}

			case types.HistoryEventTypeTaskStarted:
				executionItem.TaskStartedEventDetails = &types.TaskStartedEventDetails{
					Resource:     &stateMachineStep.Details.Resource,
					ResourceType: &stateMachineStep.Details.ResourceType,
				}

			case types.HistoryEventTypeTaskSucceeded:
				executionItem.TaskSucceededEventDetails = &types.TaskSucceededEventDetails{
					Resource:     &stateMachineStep.Details.Resource,
					ResourceType: &stateMachineStep.Details.ResourceType,
					Output:       &stateMachineStep.Details.Output,
				}

			case
				types.HistoryEventTypeTaskStateEntered,
				types.HistoryEventTypePassStateEntered,
				types.HistoryEventTypeParallelStateEntered,
				types.HistoryEventTypeMapStateEntered,
				types.HistoryEventTypeChoiceStateEntered,
				types.HistoryEventTypeSucceedStateEntered,
				types.HistoryEventTypeFailStateEntered:

				executionItem.StateEnteredEventDetails = &types.StateEnteredEventDetails{
					Input: &stateMachineStep.Details.Input,
					Name:  &stateMachineStep.Details.Name,
				}

			case
				types.HistoryEventTypeTaskStateExited,
				types.HistoryEventTypePassStateExited,
				types.HistoryEventTypeParallelStateExited,
				types.HistoryEventTypeMapStateExited,
				types.HistoryEventTypeChoiceStateExited,
				types.HistoryEventTypeSucceedStateExited:

				executionItem.StateExitedEventDetails = &types.StateExitedEventDetails{
					Output: &stateMachineStep.Details.Output,
					Name:   &stateMachineStep.Details.Name,
				}
			}
			historyEvents = append(historyEvents, executionItem)
		}

		inst.ExecutionHistory = &sfn.GetExecutionHistoryOutput{
			Events: historyEvents,
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateTable()
	})
}

func (inst *StateMachineExecutionDetailsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedState = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *StateMachineExecutionDetailsTable) GetSelectedStepInput() string {
	return inst.selectedState.Input
}

func (inst *StateMachineExecutionDetailsTable) GetSelectedStepOutput() string {
	return inst.selectedState.Output
}

func (inst *StateMachineExecutionDetailsTable) GetSelectedStepErrorCause() string {
	return inst.selectedState.Casue
}
