package sfn

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func TestSfnStateFromEvent(t *testing.T) {
	var tests = []struct {
		event types.HistoryEventType
		want  SfnStateType
	}{
		{types.HistoryEventTypeChoiceStateEntered, SfnStateTypeChoice},
		{types.HistoryEventTypeChoiceStateExited, SfnStateTypeChoice},
		{types.HistoryEventTypeFailStateEntered, SfnStateTypeFail},
		{types.HistoryEventTypeMapIterationStarted, SfnStateTypeMap},
		{types.HistoryEventTypeMapStateExited, SfnStateTypeMap},
		{types.HistoryEventTypeParallelStateFailed, SfnStateTypeParallel},
		{types.HistoryEventTypePassStateEntered, SfnStateTypePass},
		{types.HistoryEventTypeSucceedStateExited, SfnStateTypeSucceed},
		{types.HistoryEventTypeTaskScheduled, SfnStateTypeTask},
		{types.HistoryEventTypeTaskTimedOut, SfnStateTypeTask},
		{types.HistoryEventTypeWaitStateEntered, SfnStateTypeWait},
		{types.HistoryEventTypeExecutionStarted, SfnStateTypeUnknown},
		{types.HistoryEventTypeLambdaFunctionFailed, SfnStateTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			if got := SfnStateFromEvent(tt.event); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
