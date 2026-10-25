package sfn

import (
	"slices"
	"testing"
)

const orderDefinition = `{
  "Comment": "orders",
  "StartAt": "Validate",
  "States": {
    "Unused": {"Type": "Pass", "End": true},
    "Done": {"Type": "Succeed"},
    "Charge": {
      "Type": "Task", "Resource": "arn:x", "Next": "Done",
      "Catch": [{"ErrorEquals": ["States.ALL", "Boom"], "Next": "Failed"}]
    },
    "Failed": {"Type": "Fail"},
    "Route": {
      "Type": "Choice",
      "Choices": [
        {"Variable": "$.paid", "BooleanEquals": true, "Next": "Charge"},
        {"Variable": "$.kind", "StringEquals": "free", "Next": "Done"}
      ],
      "Default": "Failed"
    },
    "Validate": {"Type": "Pass", "Next": "Route"}
  }
}`

func stateNames(workflow *aslWorkflow) []string {
	var names []string
	for _, state := range workflow.States {
		names = append(names, state.Name)
	}
	return names
}

func findState(t *testing.T, workflow *aslWorkflow, name string) *aslState {
	t.Helper()
	for _, state := range workflow.States {
		if state.Name == name {
			return state
		}
	}
	t.Fatalf("state %q not found in %v", name, stateNames(workflow))
	return nil
}

func TestParseWorkflow__OrdersStatesByFlowThenDefinitionOrder(t *testing.T) {
	var workflow, err = parseWorkflow(orderDefinition)
	if err != nil {
		t.Fatal(err)
	}

	var want = []string{"Validate", "Route", "Charge", "Done", "Failed", "Unused"}
	if got := stateNames(workflow); !slices.Equal(got, want) {
		t.Errorf("states = %v, want %v", got, want)
	}
}

func TestParseWorkflow__ReadsTransitions(t *testing.T) {
	var workflow, _ = parseWorkflow(orderDefinition)

	var route = findState(t, workflow, "Route")
	var want = []aslTransition{
		{"when $.paid BooleanEquals true", "Charge"},
		{`when $.kind StringEquals "free"`, "Done"},
		{"default", "Failed"},
	}
	if !slices.Equal(route.Transitions, want) {
		t.Errorf("route transitions = %v, want %v", route.Transitions, want)
	}

	var charge = findState(t, workflow, "Charge")
	want = []aslTransition{{"next", "Done"}, {"catch States.ALL, Boom", "Failed"}}
	if !slices.Equal(charge.Transitions, want) {
		t.Errorf("charge transitions = %v, want %v", charge.Transitions, want)
	}
}

func TestParseWorkflow__MarksStartAndEndStates(t *testing.T) {
	var workflow, _ = parseWorkflow(orderDefinition)

	if !findState(t, workflow, "Validate").Start {
		t.Error("Validate should be the start state")
	}
	for _, name := range []string{"Done", "Failed", "Unused"} {
		if !findState(t, workflow, name).End {
			t.Errorf("%s should be an end state", name)
		}
	}
	if findState(t, workflow, "Route").End {
		t.Error("Route should not be an end state")
	}
}

func TestParseWorkflow__ReadsNestedWorkflows(t *testing.T) {
	var workflow, err = parseWorkflow(`{
  "StartAt": "Fan",
  "States": {
    "Fan": {"Type": "Parallel", "End": true, "Branches": [
      {"StartAt": "A", "States": {"A": {"Type": "Pass", "End": true}}},
      {"StartAt": "B", "States": {"B": {"Type": "Pass", "End": true}}}
    ]},
    "Each": {"Type": "Map", "End": true, "ItemProcessor": {
      "StartAt": "Item", "States": {"Item": {"Type": "Pass", "End": true}}
    }}
  }
}`)
	if err != nil {
		t.Fatal(err)
	}

	var fan = findState(t, workflow, "Fan")
	if len(fan.Groups) != 2 || fan.Groups[0].Label != "Branch 1" || fan.Groups[1].Label != "Branch 2" {
		t.Fatalf("fan groups = %v", fan.Groups)
	}
	if got := stateNames(fan.Groups[1].Workflow); !slices.Equal(got, []string{"B"}) {
		t.Errorf("branch 2 states = %v", got)
	}

	var each = findState(t, workflow, "Each")
	if len(each.Groups) != 1 || each.Groups[0].Label != "Item processor" {
		t.Errorf("each groups = %v", each.Groups)
	}
}

func TestParseWorkflow__RejectsDefinitionsItCannotRead(t *testing.T) {
	for _, definition := range []string{"", "not json", `{"StartAt": "A"}`, `{"States": []}`} {
		if _, err := parseWorkflow(definition); err == nil {
			t.Errorf("%q should be rejected", definition)
		}
	}
}

func TestParseWorkflow__StartAtAStateThatDoesNotExist(t *testing.T) {
	var workflow, err = parseWorkflow(`{"StartAt": "Gone", "States": {"A": {"Type": "Pass", "End": true}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := stateNames(workflow); !slices.Equal(got, []string{"A"}) {
		t.Errorf("states = %v, want [A]", got)
	}
}
