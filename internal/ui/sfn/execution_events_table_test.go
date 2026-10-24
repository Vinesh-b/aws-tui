package sfn

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newEventsTable(state StateDetails) *SfnExecutionStateEventsTable {
	var table = NewSfnExecutionStatesTable(uitest.NewAppContext(), nil)
	table.RefreshExecutionState(state)
	table.SetSelectionChangedFunc(func(row, column int) {})
	return table
}

func newEventDetails(name string, startSec, endSec int) EventDetails {
	return EventDetails{
		Name:         name,
		Type:         "TaskSucceeded",
		Resource:     "invoke",
		ResourceType: "lambda",
		StartTime:    *at(startSec),
		EndTime:      *at(endSec),
		Errors:       "",
		Input:        "in-" + name,
		Output:       "out-" + name,
		Casue:        "cause-" + name,
	}
}

var eventHeadings = core.TableRow{"StateName", "Type", "Resource", "Action", "StartTime", "Duration", "Errors"}

func TestEventsTable__EmptyStateShowsOnlyHeadings(t *testing.T) {
	var table = newEventsTable(StateDetails{})

	uitest.AssertTableRows(t, table, []core.TableRow{eventHeadings})
}

func TestEventsTable__ShowsOneRowPerEvent(t *testing.T) {
	var failing = newEventDetails("second", 5, 5)
	failing.Errors = "Boom"
	var table = newEventsTable(StateDetails{
		Events: []EventDetails{newEventDetails("first", 0, 2), failing},
	})

	uitest.AssertTableRows(t, table, []core.TableRow{
		eventHeadings,
		{"first", "TaskSucceeded", "lambda", "invoke", "2024-01-02 03:04:05", "2s", ""},
		{"second", "TaskSucceeded", "lambda", "invoke", "2024-01-02 03:04:10", "0s", "Boom"},
	})
}

func TestEventsTable__SelectedStepExposesInputOutputAndCause(t *testing.T) {
	var table = newEventsTable(StateDetails{
		Events: []EventDetails{newEventDetails("first", 0, 1), newEventDetails("second", 1, 2)},
	})

	table.Select(2, 0)

	if got := table.GetSelectedStepInput(); got != "in-second" {
		t.Errorf("input = %q, want %q", got, "in-second")
	}
	if got := table.GetSelectedStepOutput(); got != "out-second" {
		t.Errorf("output = %q, want %q", got, "out-second")
	}
	if got := table.GetSelectedStepErrorCause(); got != "cause-second" {
		t.Errorf("cause = %q, want %q", got, "cause-second")
	}
}

func TestEventsTable__SelectingHeadingKeepsPreviousStep(t *testing.T) {
	var table = newEventsTable(StateDetails{
		Events: []EventDetails{newEventDetails("first", 0, 1), newEventDetails("second", 1, 2)},
	})
	table.Select(2, 0)

	table.Select(0, 0)

	if got := table.GetSelectedStepInput(); got != "in-second" {
		t.Errorf("input = %q, want %q", got, "in-second")
	}
}
