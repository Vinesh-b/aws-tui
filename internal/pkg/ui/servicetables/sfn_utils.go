package servicetables

import (
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type SfnStateType string

// Enum values for SfnStateType
const (
	SfnStateTypeChoice   SfnStateType = "Choice"
	SfnStateTypeFail     SfnStateType = "Fail"
	SfnStateTypeMap      SfnStateType = "Map"
	SfnStateTypeParallel SfnStateType = "Parallel"
	SfnStateTypePass     SfnStateType = "Pass"
	SfnStateTypeSucceed  SfnStateType = "Succeed"
	SfnStateTypeTask     SfnStateType = "Task"
	SfnStateTypeWait     SfnStateType = "Wait"
	SfnStateTypeUnknown  SfnStateType = "Unknown"
)

func SfnStateFromEvent(eventType types.HistoryEventType) SfnStateType {
	switch eventType {
	case
		types.HistoryEventTypeChoiceStateEntered,
		types.HistoryEventTypeChoiceStateExited:
		return SfnStateTypeChoice
	case
		types.HistoryEventTypeFailStateEntered:
		return SfnStateTypeFail
	case
		types.HistoryEventTypeMapIterationAborted,
		types.HistoryEventTypeMapIterationFailed,
		types.HistoryEventTypeMapIterationStarted,
		types.HistoryEventTypeMapIterationSucceeded,
		types.HistoryEventTypeMapStateAborted,
		types.HistoryEventTypeMapStateEntered,
		types.HistoryEventTypeMapStateExited,
		types.HistoryEventTypeMapStateFailed,
		types.HistoryEventTypeMapStateStarted,
		types.HistoryEventTypeMapStateSucceeded:
		return SfnStateTypeMap
	case
		types.HistoryEventTypeParallelStateAborted,
		types.HistoryEventTypeParallelStateEntered,
		types.HistoryEventTypeParallelStateExited,
		types.HistoryEventTypeParallelStateFailed,
		types.HistoryEventTypeParallelStateStarted,
		types.HistoryEventTypeParallelStateSucceeded:
		return SfnStateTypeParallel
	case
		types.HistoryEventTypePassStateEntered,
		types.HistoryEventTypePassStateExited:
		return SfnStateTypePass
	case
		types.HistoryEventTypeSucceedStateEntered,
		types.HistoryEventTypeSucceedStateExited:
		return SfnStateTypeSucceed
	case
		types.HistoryEventTypeTaskFailed,
		types.HistoryEventTypeTaskScheduled,
		types.HistoryEventTypeTaskStarted,
		types.HistoryEventTypeTaskStartFailed,
		types.HistoryEventTypeTaskStateAborted,
		types.HistoryEventTypeTaskStateEntered,
		types.HistoryEventTypeTaskStateExited,
		types.HistoryEventTypeTaskSubmitFailed,
		types.HistoryEventTypeTaskSubmitted,
		types.HistoryEventTypeTaskSucceeded,
		types.HistoryEventTypeTaskTimedOut:
		return SfnStateTypeTask
	case
		types.HistoryEventTypeWaitStateAborted,
		types.HistoryEventTypeWaitStateEntered,
		types.HistoryEventTypeWaitStateExited:
		return SfnStateTypeWait

	}

	return SfnStateTypeUnknown
}
