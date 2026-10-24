package servicetables

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	"github.com/gdamore/tcell/v2"
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

type SfnExecutionStatesTable struct {
	*core.SelectableTable[StateDetails]
	ExecutionHistory     *sfn.GetExecutionHistoryOutput
	States               []StateDetails
	events               []EventDetails
	selectedExecutionArn string
	selectedState        StateDetails
	appCtx               *core.AppContext
	api                  *awsapi.StateMachineApi
	cwlApi               *awsapi.CloudWatchLogsApi
}

func NewSfnExecutionDetailsTable(
	appCtx *core.AppContext,
	api *awsapi.StateMachineApi,
	cwlApi *awsapi.CloudWatchLogsApi,
) *SfnExecutionStatesTable {

	var view = &SfnExecutionStatesTable{
		SelectableTable: core.NewSelectableTable[StateDetails](
			"Execution Details",
			core.TableRow{
				"Name",
				"Type",
				"Duration",
			},
			appCtx,
		),
		ExecutionHistory:     nil,
		selectedExecutionArn: "",
		selectedState:        StateDetails{},

		appCtx: appCtx,
		api:    api,
		cwlApi: cwlApi,
	}

	view.populateTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			view.RefreshExecutionStates(view.selectedExecutionArn, true)
			return nil
		}
		return event
	})

	return view
}

type EventDetails struct {
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

func (e *EventDetails) setError(err, cause *string) {
	e.Errors = aws.ToString(err)
	e.Casue = aws.ToString(cause)
}

func (e *EventDetails) setResource(resource, resourceType *string) {
	e.Resource = aws.ToString(resource)
	e.ResourceType = aws.ToString(resourceType)
}

func (inst *SfnExecutionStatesTable) parseExecutionHistory() []EventDetails {
	var results []EventDetails
	if inst.ExecutionHistory == nil {
		return results
	}

	var executionStartTime time.Time
	var taskStartTime time.Time
	for _, row := range inst.ExecutionHistory.Events {
		var timestamp = aws.ToTime(row.Timestamp)
		var details = EventDetails{
			Id:        row.Id,
			Type:      string(row.Type),
			StartTime: timestamp,
			EndTime:   timestamp,
		}

		switch row.Type {
		case types.HistoryEventTypeExecutionStarted:
			details.Input = aws.ToString(row.ExecutionStartedEventDetails.Input)
			executionStartTime = timestamp

		case types.HistoryEventTypeExecutionSucceeded:
			details.Output = aws.ToString(row.ExecutionSucceededEventDetails.Output)
			details.StartTime = executionStartTime

		case types.HistoryEventTypeExecutionFailed:
			details.setError(row.ExecutionFailedEventDetails.Error, row.ExecutionFailedEventDetails.Cause)
			details.StartTime = executionStartTime

		case types.HistoryEventTypeExecutionAborted:
			details.setError(row.ExecutionAbortedEventDetails.Error, row.ExecutionAbortedEventDetails.Cause)
			details.StartTime = executionStartTime

		case types.HistoryEventTypeExecutionTimedOut:
			details.setError(row.ExecutionTimedOutEventDetails.Error, row.ExecutionTimedOutEventDetails.Cause)
			details.StartTime = executionStartTime

		case types.HistoryEventTypeTaskStartFailed:
			details.setError(row.TaskStartFailedEventDetails.Error, row.TaskStartFailedEventDetails.Cause)

		case types.HistoryEventTypeTaskFailed:
			details.setError(row.TaskFailedEventDetails.Error, row.TaskFailedEventDetails.Cause)

		case types.HistoryEventTypeTaskScheduled:
			details.setResource(row.TaskScheduledEventDetails.Resource, row.TaskScheduledEventDetails.ResourceType)
			details.Input = aws.ToString(row.TaskScheduledEventDetails.Parameters)

		case types.HistoryEventTypeTaskSubmitted:
			details.setResource(row.TaskSubmittedEventDetails.Resource, row.TaskSubmittedEventDetails.ResourceType)
			details.Output = aws.ToString(row.TaskSubmittedEventDetails.Output)

		case types.HistoryEventTypeTaskStarted:
			details.setResource(row.TaskStartedEventDetails.Resource, row.TaskStartedEventDetails.ResourceType)

		case types.HistoryEventTypeTaskSucceeded:
			details.setResource(row.TaskSucceededEventDetails.Resource, row.TaskSucceededEventDetails.ResourceType)
			details.Output = aws.ToString(row.TaskSucceededEventDetails.Output)

		case
			types.HistoryEventTypeTaskStateEntered,
			types.HistoryEventTypePassStateEntered,
			types.HistoryEventTypeParallelStateEntered,
			types.HistoryEventTypeMapStateEntered,
			types.HistoryEventTypeChoiceStateEntered,
			types.HistoryEventTypeWaitStateEntered,
			types.HistoryEventTypeSucceedStateEntered,
			types.HistoryEventTypeFailStateEntered:

			details.Name = aws.ToString(row.StateEnteredEventDetails.Name)
			details.Input = aws.ToString(row.StateEnteredEventDetails.Input)
			taskStartTime = timestamp

		case
			types.HistoryEventTypeTaskStateExited,
			types.HistoryEventTypePassStateExited,
			types.HistoryEventTypeParallelStateExited,
			types.HistoryEventTypeMapStateExited,
			types.HistoryEventTypeChoiceStateExited,
			types.HistoryEventTypeWaitStateExited,
			types.HistoryEventTypeSucceedStateExited:

			var exitedName = aws.ToString(row.StateExitedEventDetails.Name)
			var enteredBefore = slices.ContainsFunc(results, func(d EventDetails) bool {
				return d.Name == exitedName
			})

			if enteredBefore {
				details.Output = aws.ToString(row.StateExitedEventDetails.Output)
				details.StartTime = taskStartTime
			}
		}

		results = append(results, details)
	}
	inst.events = results

	return results
}

func (inst *SfnExecutionStatesTable) populateTable() {
	var tableData []core.TableRow

	for _, row := range inst.States {
		tableData = append(tableData, core.TableRow{
			row.Name,
			string(row.Type),
			row.Duration.String(),
		})
	}

	inst.SetData(tableData, inst.States, 0)
	inst.Select(1, 0)
}

func (inst *SfnExecutionStatesTable) reloadFromHistory() {
	inst.parseExecutionHistory()
	inst.parseStates()
	inst.populateTable()
}

func (inst *SfnExecutionStatesTable) RefreshExecutionStates(executionArn string, force bool) {
	inst.selectedExecutionArn = executionArn
	var dataLoader = core.NewUiDataLoader(inst.appCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error = nil
		inst.ExecutionHistory, err = inst.api.GetExecutionHistory(executionArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, inst.reloadFromHistory)
}

func (inst *SfnExecutionStatesTable) RefreshExpressExecutionStates(executionItem ExecutionItem, force bool) {
	inst.selectedExecutionArn = aws.ToString(executionItem.ExecutionArn)
	var findExecutionDetailsQuery = fmt.Sprintf(
		`fields @message | filter execution_arn="%s" | sort id asc | limit 1000`,
		inst.selectedExecutionArn,
	)

	var insightsQuery = InsightsQuery{
		query:     findExecutionDetailsQuery,
		startTime: aws.ToTime(executionItem.StartDate),
		endTime:   aws.ToTime(executionItem.StopDate),
	}

	var insightsQueryRunner = NewInsightsQueryRunner(inst.appCtx.App, inst.cwlApi)
	insightsQueryRunner.ErrorMessageCallback = inst.ErrorMessageCallback

	var resultsChan = make(chan [][]cwlTypes.ResultField)

	var dataLoader = core.NewUiDataLoader(inst.appCtx.App, 10)
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
			historyEvents = append(historyEvents, historyEventFromStep(stateMachineStep))
		}

		inst.ExecutionHistory = &sfn.GetExecutionHistoryOutput{
			Events: historyEvents,
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, inst.reloadFromHistory)
}

func historyEventFromStep(stateMachineStep StateMachineStep) types.HistoryEvent {
	var event = types.HistoryEvent{
		Id:              stateMachineStep.Id,
		PreviousEventId: stateMachineStep.PreviousId,
		Timestamp:       aws.Time(time.UnixMilli(stateMachineStep.Timestamp)),
		Type:            stateMachineStep.StateType,
	}

	switch stateMachineStep.StateType {
	case types.HistoryEventTypeExecutionStarted:
		event.ExecutionStartedEventDetails = &types.ExecutionStartedEventDetails{
			Input: &stateMachineStep.Details.Input,
		}

	case types.HistoryEventTypeExecutionSucceeded:
		event.ExecutionSucceededEventDetails = &types.ExecutionSucceededEventDetails{
			Output: &stateMachineStep.Details.Output,
		}

	case types.HistoryEventTypeExecutionFailed:
		event.ExecutionFailedEventDetails = &types.ExecutionFailedEventDetails{
			Error: &stateMachineStep.Details.ErrorCode,
			Cause: &stateMachineStep.Details.ErrorCause,
		}

	case types.HistoryEventTypeExecutionAborted:
		event.ExecutionAbortedEventDetails = &types.ExecutionAbortedEventDetails{
			Error: &stateMachineStep.Details.ErrorCode,
			Cause: &stateMachineStep.Details.ErrorCause,
		}

	case types.HistoryEventTypeExecutionTimedOut:
		event.ExecutionTimedOutEventDetails = &types.ExecutionTimedOutEventDetails{
			Error: &stateMachineStep.Details.ErrorCode,
			Cause: &stateMachineStep.Details.ErrorCause,
		}

	case types.HistoryEventTypeTaskStartFailed:
		event.TaskStartFailedEventDetails = &types.TaskStartFailedEventDetails{
			Error: &stateMachineStep.Details.ErrorCode,
			Cause: &stateMachineStep.Details.ErrorCause,
		}

	case types.HistoryEventTypeTaskFailed:
		event.TaskFailedEventDetails = &types.TaskFailedEventDetails{
			Error: &stateMachineStep.Details.ErrorCode,
			Cause: &stateMachineStep.Details.ErrorCause,
		}

	case types.HistoryEventTypeTaskScheduled:
		event.TaskScheduledEventDetails = &types.TaskScheduledEventDetails{
			Resource:     &stateMachineStep.Details.Resource,
			ResourceType: &stateMachineStep.Details.ResourceType,
			Parameters:   &stateMachineStep.Details.Parameters,
		}

	case types.HistoryEventTypeTaskSubmitted:
		event.TaskSubmittedEventDetails = &types.TaskSubmittedEventDetails{
			Resource:     &stateMachineStep.Details.Resource,
			ResourceType: &stateMachineStep.Details.ResourceType,
			Output:       &stateMachineStep.Details.Output,
		}

	case types.HistoryEventTypeTaskStarted:
		event.TaskStartedEventDetails = &types.TaskStartedEventDetails{
			Resource:     &stateMachineStep.Details.Resource,
			ResourceType: &stateMachineStep.Details.ResourceType,
		}

	case types.HistoryEventTypeTaskSucceeded:
		event.TaskSucceededEventDetails = &types.TaskSucceededEventDetails{
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

		event.StateEnteredEventDetails = &types.StateEnteredEventDetails{
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

		event.StateExitedEventDetails = &types.StateExitedEventDetails{
			Output: &stateMachineStep.Details.Output,
			Name:   &stateMachineStep.Details.Name,
		}
	}
	return event
}

func (inst *SfnExecutionStatesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedState = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SfnExecutionStatesTable) parseStates() []StateDetails {
	var results []StateDetails

	var currentState *StateDetails = nil
	for _, e := range inst.events {
		var eventStateType = SfnStateFromEvent(types.HistoryEventType(e.Type))

		// Only StateEntered events have the name property set
		if len(e.Name) > 0 {
			results = append(results, StateDetails{
				Name: e.Name,
				Id:   e.Id,
				Type: eventStateType,
			})
			currentState = &results[len(results)-1]
		}

		if currentState != nil && eventStateType == currentState.Type {
			currentState.Events = append(currentState.Events, e)
		}
	}

	for i, s := range results {
		var eventCount = len(s.Events)
		results[i].Duration = s.Events[eventCount-1].EndTime.Sub(s.Events[0].StartTime)
	}

	inst.States = results
	return results
}

func (inst *SfnExecutionStatesTable) GetSelectedState() StateDetails {
	return inst.selectedState
}
