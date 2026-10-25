package sfn

import (
	"strings"
	"time"
)

type stateOutcome int

// The outcomes are ordered from least to most important: when a state ran more
// than once the highest one is the one shown.
const (
	outcomeNotReached stateOutcome = iota
	outcomeSucceeded
	outcomeRunning
	outcomeFailed
)

// stateRun is what an execution did in one state, over all the times the
// state ran, which is more than once in a loop or a Map state.
type stateRun struct {
	Outcome  stateOutcome
	Duration time.Duration
	Count    int
}

// outcomeOf works out how one run of a state ended from its events. A state
// that was left succeeded, even if a task in it failed before a retry or a
// catch handled the failure. One that was not left and has an error, or is a
// Fail state, failed. Any other state is still running.
func outcomeOf(state StateDetails) stateOutcome {
	var left, failed bool
	for _, event := range state.Events {
		switch {
		case strings.HasSuffix(event.Type, "StateExited"):
			left = true
		case event.Errors != "",
			strings.Contains(event.Type, "Failed"),
			strings.Contains(event.Type, "TimedOut"),
			strings.Contains(event.Type, "Aborted"):
			failed = true
		}
	}

	switch {
	case state.Type == SfnStateTypeSucceed:
		return outcomeSucceeded
	case state.Type == SfnStateTypeFail:
		return outcomeFailed
	case left:
		return outcomeSucceeded
	case failed:
		return outcomeFailed
	default:
		return outcomeRunning
	}
}

// summariseStates groups the states an execution went through by name. State
// names are unique within a state machine, including its nested workflows.
func summariseStates(states []StateDetails) map[string]stateRun {
	var runs = map[string]stateRun{}
	for _, state := range states {
		var run = runs[state.Name]
		run.Count++
		run.Duration += state.Duration
		run.Outcome = max(run.Outcome, outcomeOf(state))
		runs[state.Name] = run
	}

	return runs
}
