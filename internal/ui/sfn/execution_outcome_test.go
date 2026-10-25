package sfn

import (
	"testing"
	"time"
)

func stateWith(name string, stateType SfnStateType, seconds int, eventTypes ...string) StateDetails {
	var state = StateDetails{
		Name: name, Type: stateType, Duration: time.Duration(seconds) * time.Second,
	}
	for _, eventType := range eventTypes {
		state.Events = append(state.Events, EventDetails{Type: eventType})
	}
	return state
}

func TestOutcomeOf__ClassifiesHowAStateEnded(t *testing.T) {
	var failedTask = stateWith("t", SfnStateTypeTask, 1, "TaskStateEntered", "TaskFailed")
	failedTask.Events[1].Errors = "Boom"

	var tests = []struct {
		name  string
		state StateDetails
		want  stateOutcome
	}{
		{"left", stateWith("a", SfnStateTypeTask, 1, "TaskStateEntered", "TaskSucceeded", "TaskStateExited"), outcomeSucceeded},
		{"never left", stateWith("a", SfnStateTypeTask, 1, "TaskStateEntered", "TaskScheduled"), outcomeRunning},
		{"failed and not left", failedTask, outcomeFailed},
		{"failed event type", stateWith("a", SfnStateTypeTask, 1, "TaskStateEntered", "TaskTimedOut"), outcomeFailed},
		{"failed but retried then left", stateWith("a", SfnStateTypeTask, 1, "TaskStateEntered", "TaskFailed", "TaskStateExited"), outcomeSucceeded},
		{"fail state", stateWith("f", SfnStateTypeFail, 0, "FailStateEntered"), outcomeFailed},
		{"succeed state", stateWith("s", SfnStateTypeSucceed, 0, "SucceedStateEntered"), outcomeSucceeded},
	}

	for _, test := range tests {
		if got := outcomeOf(test.state); got != test.want {
			t.Errorf("%s: outcome = %v, want %v", test.name, got, test.want)
		}
	}
}

func TestSummariseStates__CombinesRepeatedStates(t *testing.T) {
	var runs = summariseStates([]StateDetails{
		stateWith("loop", SfnStateTypeTask, 2, "TaskStateEntered", "TaskStateExited"),
		stateWith("other", SfnStateTypePass, 1, "PassStateEntered", "PassStateExited"),
		stateWith("loop", SfnStateTypeTask, 3, "TaskStateEntered"),
	})

	var loop = runs["loop"]
	if loop.Count != 2 || loop.Duration != 5*time.Second || loop.Outcome != outcomeRunning {
		t.Errorf("loop = %+v, want 2 runs over 5s, still running", loop)
	}
	if other := runs["other"]; other.Count != 1 || other.Outcome != outcomeSucceeded {
		t.Errorf("other = %+v", other)
	}
	if _, ok := runs["missing"]; ok {
		t.Error("a state that never ran should have no run")
	}
}
