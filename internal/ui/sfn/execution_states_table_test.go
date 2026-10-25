package sfn

import (
	"encoding/json"
	"testing"
	"time"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func newStatesTable(events ...types.HistoryEvent) *SfnExecutionStatesTable {
	var table = NewSfnExecutionDetailsTable(uitest.NewAppContext(), nil, nil)
	table.ExecutionHistory = &sfn.GetExecutionHistoryOutput{Events: events}
	return table
}

func historyEvent(id int64, eventType types.HistoryEventType, seconds int) types.HistoryEvent {
	return types.HistoryEvent{Id: id, Type: eventType, Timestamp: at(seconds)}
}

func stateEntered(id int64, eventType types.HistoryEventType, seconds int, name, input string) types.HistoryEvent {
	var event = historyEvent(id, eventType, seconds)
	event.StateEnteredEventDetails = &types.StateEnteredEventDetails{
		Name: aws.String(name), Input: aws.String(input),
	}
	return event
}

func stateExited(id int64, eventType types.HistoryEventType, seconds int, name, output string) types.HistoryEvent {
	var event = historyEvent(id, eventType, seconds)
	event.StateExitedEventDetails = &types.StateExitedEventDetails{
		Name: aws.String(name), Output: aws.String(output),
	}
	return event
}

func executionStarted(seconds int, input string) types.HistoryEvent {
	var event = historyEvent(1, types.HistoryEventTypeExecutionStarted, seconds)
	event.ExecutionStartedEventDetails = &types.ExecutionStartedEventDetails{Input: aws.String(input)}
	return event
}

func TestParseExecutionHistory__NoHistoryGivesNoEvents(t *testing.T) {
	var table = NewSfnExecutionDetailsTable(uitest.NewAppContext(), nil, nil)

	if got := table.parseExecutionHistory(); len(got) != 0 {
		t.Errorf("events = %v, want none", got)
	}
}

func TestParseExecutionHistory__ExecutionStartedKeepsInput(t *testing.T) {
	var table = newStatesTable(executionStarted(0, `{"order":1}`))

	var events = table.parseExecutionHistory()

	if len(events) != 1 || events[0].Input != `{"order":1}` || events[0].Type != "ExecutionStarted" {
		t.Errorf("events = %+v, want one ExecutionStarted with input", events)
	}
}

func TestParseExecutionHistory__ExecutionEndEventsSpanFromExecutionStart(t *testing.T) {
	var succeeded = historyEvent(2, types.HistoryEventTypeExecutionSucceeded, 50)
	succeeded.ExecutionSucceededEventDetails = &types.ExecutionSucceededEventDetails{Output: aws.String("done")}
	var failed = historyEvent(3, types.HistoryEventTypeExecutionFailed, 60)
	failed.ExecutionFailedEventDetails = &types.ExecutionFailedEventDetails{
		Error: aws.String("Boom"), Cause: aws.String("bad input"),
	}
	var table = newStatesTable(executionStarted(10, "{}"), succeeded, failed)

	var events = table.parseExecutionHistory()

	var tests = []struct {
		name      string
		event     EventDetails
		wantStart time.Time
		wantEnd   time.Time
	}{
		{"succeeded", events[1], *at(10), *at(50)},
		{"failed", events[2], *at(10), *at(60)},
	}
	for _, tt := range tests {
		if !tt.event.StartTime.Equal(tt.wantStart) || !tt.event.EndTime.Equal(tt.wantEnd) {
			t.Errorf("%s spans %v - %v, want %v - %v",
				tt.name, tt.event.StartTime, tt.event.EndTime, tt.wantStart, tt.wantEnd)
		}
	}
	if events[1].Output != "done" {
		t.Errorf("output = %q, want %q", events[1].Output, "done")
	}
	if events[2].Errors != "Boom" || events[2].Cause != "bad input" {
		t.Errorf("error, cause = %q, %q, want Boom, bad input", events[2].Errors, events[2].Cause)
	}
}

func TestParseExecutionHistory__TaskEventsCarryResourceAndPayloads(t *testing.T) {
	var scheduled = historyEvent(3, types.HistoryEventTypeTaskScheduled, 11)
	scheduled.TaskScheduledEventDetails = &types.TaskScheduledEventDetails{
		Resource: aws.String("invoke"), ResourceType: aws.String("lambda"), Parameters: aws.String(`{"p":1}`),
	}
	var succeeded = historyEvent(4, types.HistoryEventTypeTaskSucceeded, 12)
	succeeded.TaskSucceededEventDetails = &types.TaskSucceededEventDetails{
		Resource: aws.String("invoke"), ResourceType: aws.String("lambda"), Output: aws.String(`{"ok":true}`),
	}
	var taskFailed = historyEvent(5, types.HistoryEventTypeTaskFailed, 13)
	taskFailed.TaskFailedEventDetails = &types.TaskFailedEventDetails{
		Error: aws.String("Timeout"), Cause: aws.String("took too long"),
	}
	var table = newStatesTable(scheduled, succeeded, taskFailed)

	var events = table.parseExecutionHistory()

	if events[0].Resource != "invoke" || events[0].ResourceType != "lambda" || events[0].Input != `{"p":1}` {
		t.Errorf("scheduled = %+v", events[0])
	}
	if events[1].Output != `{"ok":true}` || events[1].Resource != "invoke" {
		t.Errorf("succeeded = %+v", events[1])
	}
	if events[2].Errors != "Timeout" || events[2].Cause != "took too long" {
		t.Errorf("failed = %+v", events[2])
	}
}

func TestParseExecutionHistory__StateExitSpansFromStateEntry(t *testing.T) {
	var table = newStatesTable(
		executionStarted(0, "{}"),
		stateEntered(2, types.HistoryEventTypeTaskStateEntered, 10, "Charge", `{"in":1}`),
		stateExited(3, types.HistoryEventTypeTaskStateExited, 40, "Charge", `{"out":2}`),
	)

	var events = table.parseExecutionHistory()

	var exited = events[2]
	if !exited.StartTime.Equal(*at(10)) || !exited.EndTime.Equal(*at(40)) {
		t.Errorf("exit spans %v - %v, want %v - %v", exited.StartTime, exited.EndTime, *at(10), *at(40))
	}
	if exited.Output != `{"out":2}` {
		t.Errorf("output = %q, want %q", exited.Output, `{"out":2}`)
	}
}

func TestParseExecutionHistory__ExitWithoutMatchingEntryKeepsOwnTimestamp(t *testing.T) {
	var table = newStatesTable(
		stateExited(2, types.HistoryEventTypeTaskStateExited, 40, "Orphan", "ignored"),
	)

	var exited = table.parseExecutionHistory()[0]

	if exited.Output != "" || !exited.StartTime.Equal(*at(40)) {
		t.Errorf("exit = %+v, want empty output and own timestamp", exited)
	}
}

func TestParseStates__GroupsEventsUnderEachEnteredState(t *testing.T) {
	var scheduled = historyEvent(3, types.HistoryEventTypeTaskScheduled, 11)
	scheduled.TaskScheduledEventDetails = &types.TaskScheduledEventDetails{}
	var table = newStatesTable(
		executionStarted(0, "{}"),
		stateEntered(2, types.HistoryEventTypeTaskStateEntered, 10, "Charge", "{}"),
		scheduled,
		stateExited(4, types.HistoryEventTypeTaskStateExited, 40, "Charge", "{}"),
		stateEntered(5, types.HistoryEventTypePassStateEntered, 41, "Notify", "{}"),
		stateExited(6, types.HistoryEventTypePassStateExited, 45, "Notify", "{}"),
	)
	table.parseExecutionHistory()

	var states = table.parseStates()

	if len(states) != 2 {
		t.Fatalf("states = %d, want 2", len(states))
	}
	var tests = []struct {
		state      StateDetails
		wantName   string
		wantType   SfnStateType
		wantEvents int
		wantTime   time.Duration
	}{
		{states[0], "Charge", SfnStateTypeTask, 3, 30 * time.Second},
		{states[1], "Notify", SfnStateTypePass, 2, 4 * time.Second},
	}
	for _, tt := range tests {
		if tt.state.Name != tt.wantName || tt.state.Type != tt.wantType ||
			len(tt.state.Events) != tt.wantEvents || tt.state.Duration != tt.wantTime {
			t.Errorf("state = {%s %s events=%d %v}, want {%s %s events=%d %v}",
				tt.state.Name, tt.state.Type, len(tt.state.Events), tt.state.Duration,
				tt.wantName, tt.wantType, tt.wantEvents, tt.wantTime)
		}
	}
}

func TestParseStates__NoStatesWithoutStateEnteredEvents(t *testing.T) {
	var table = newStatesTable(executionStarted(0, "{}"))
	table.parseExecutionHistory()

	if states := table.parseStates(); len(states) != 0 {
		t.Errorf("states = %v, want none", states)
	}
}

func TestStatesTable__ShowsHeadingsAndOneRowPerState(t *testing.T) {
	var table = newStatesTable(
		stateEntered(2, types.HistoryEventTypeTaskStateEntered, 10, "Charge", "{}"),
		stateExited(3, types.HistoryEventTypeTaskStateExited, 100, "Charge", "{}"),
	)

	table.reloadFromHistory()

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "Type", "Duration"},
		{"Charge", "Task", "1m30s"},
	})
}

func TestStatesTable__SelectingStateTracksIt(t *testing.T) {
	var table = newStatesTable(
		stateEntered(2, types.HistoryEventTypeTaskStateEntered, 10, "Charge", "{}"),
		stateExited(3, types.HistoryEventTypeTaskStateExited, 20, "Charge", "{}"),
		stateEntered(4, types.HistoryEventTypePassStateEntered, 21, "Notify", "{}"),
		stateExited(5, types.HistoryEventTypePassStateExited, 22, "Notify", "{}"),
	)
	table.reloadFromHistory()
	table.SetSelectionChangedFunc(func(row, column int) {})

	table.Select(2, 0)

	if got := table.GetSelectedState().Name; got != "Notify" {
		t.Errorf("selected state = %q, want %q", got, "Notify")
	}
}

func TestHistoryEventFromStep__MapsStepPayloadsToEventDetails(t *testing.T) {
	var step = StateMachineStep{Id: 7, PreviousId: 6, Timestamp: baseTime.UnixMilli()}
	step.Details.Input = "in"
	step.Details.Output = "out"
	step.Details.Name = "Charge"
	step.Details.ErrorCode = "Boom"
	step.Details.ErrorCause = "bad"

	var tests = []struct {
		eventType types.HistoryEventType
		check     func(types.HistoryEvent) bool
	}{
		{types.HistoryEventTypeExecutionStarted, func(e types.HistoryEvent) bool {
			return aws.ToString(e.ExecutionStartedEventDetails.Input) == "in"
		}},
		{types.HistoryEventTypeExecutionSucceeded, func(e types.HistoryEvent) bool {
			return aws.ToString(e.ExecutionSucceededEventDetails.Output) == "out"
		}},
		{types.HistoryEventTypeExecutionFailed, func(e types.HistoryEvent) bool {
			return aws.ToString(e.ExecutionFailedEventDetails.Error) == "Boom" &&
				aws.ToString(e.ExecutionFailedEventDetails.Cause) == "bad"
		}},
		{types.HistoryEventTypeTaskFailed, func(e types.HistoryEvent) bool {
			return aws.ToString(e.TaskFailedEventDetails.Error) == "Boom"
		}},
		{types.HistoryEventTypeTaskStateEntered, func(e types.HistoryEvent) bool {
			return aws.ToString(e.StateEnteredEventDetails.Name) == "Charge" &&
				aws.ToString(e.StateEnteredEventDetails.Input) == "in"
		}},
		{types.HistoryEventTypeTaskStateExited, func(e types.HistoryEvent) bool {
			return aws.ToString(e.StateExitedEventDetails.Name) == "Charge" &&
				aws.ToString(e.StateExitedEventDetails.Output) == "out"
		}},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			step.StateType = tt.eventType

			var event = historyEventFromStep(step)

			if event.Id != 7 || event.PreviousEventId != 6 || event.Type != tt.eventType {
				t.Errorf("event header = %d/%d/%s", event.Id, event.PreviousEventId, event.Type)
			}
			if !event.Timestamp.Equal(baseTime) {
				t.Errorf("timestamp = %v, want %v", event.Timestamp, baseTime)
			}
			if !tt.check(event) {
				t.Errorf("event details not mapped: %+v", event)
			}
		})
	}
}

func TestStateMachineStep__DecodesLogEntry(t *testing.T) {
	var raw = `{
		"id": "3", "previous_event_id": "2", "event_timestamp": "1700000000000",
		"type": "TaskStateEntered", "execution_arn": "arn:exec",
		"details": {"name": "Charge", "input": "{}"}
	}`

	var step StateMachineStep
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if step.Id != 3 || step.PreviousId != 2 || step.Timestamp != 1700000000000 ||
		step.StateType != types.HistoryEventTypeTaskStateEntered || step.Details.Name != "Charge" {
		t.Errorf("step = %+v", step)
	}
}

func TestStatesTable__ReportsTheStatesWhenShown(t *testing.T) {
	var table = newStatesTable(
		stateEntered(2, types.HistoryEventTypePassStateEntered, 1, "First", "{}"),
		stateExited(3, types.HistoryEventTypePassStateExited, 2, "First", "{}"),
	)
	var names []string
	table.SetStatesLoadedFunc(func(states []StateDetails) {
		for _, state := range states {
			names = append(names, state.Name)
		}
	})

	table.reloadFromHistory()

	if len(names) != 1 || names[0] != "First" {
		t.Errorf("loaded func saw %v, want [First]", names)
	}
}
