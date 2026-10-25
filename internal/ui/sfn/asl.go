package sfn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"aws-tui/internal/utils"
)

// The model below is the part of an Amazon States Language definition needed
// to show how a state machine flows: the states, where each one can go next,
// and the workflows nested in Parallel and Map states.

type aslTransition struct {
	Label  string
	Target string
}

type aslGroup struct {
	Label    string
	Workflow *aslWorkflow
}

type aslState struct {
	Name        string
	Type        string
	Comment     string
	Start       bool
	End         bool
	Transitions []aslTransition
	Groups      []aslGroup
}

type aslWorkflow struct {
	Comment string
	StartAt string
	// States are in flow order: from StartAt along the transitions, then any
	// state that cannot be reached in the order they are defined.
	States []*aslState
}

type rawChoice map[string]json.RawMessage

type rawCatch struct {
	ErrorEquals []string
	Next        string
}

type rawState struct {
	Type          string
	Comment       string
	Next          string
	Default       string
	End           bool
	Choices       []rawChoice
	Catch         []rawCatch
	Branches      []rawWorkflow
	ItemProcessor *rawWorkflow
	Iterator      *rawWorkflow
}

// rawStates keeps the order the states are written in, which a map does not.
type rawStates struct {
	names  []string
	states map[string]rawState
}

func (inst *rawStates) UnmarshalJSON(data []byte) error {
	var dec = json.NewDecoder(bytes.NewReader(data))
	if _, err := dec.Token(); err != nil {
		return err
	}

	inst.names = nil
	inst.states = map[string]rawState{}
	for dec.More() {
		var nameToken, err = dec.Token()
		if err != nil {
			return err
		}
		var name, ok = nameToken.(string)
		if !ok {
			return fmt.Errorf("unexpected state name %v", nameToken)
		}

		var state rawState
		if err := dec.Decode(&state); err != nil {
			return fmt.Errorf("state %q: %w", name, err)
		}
		inst.names = append(inst.names, name)
		inst.states[name] = state
	}

	return nil
}

type rawWorkflow struct {
	Comment string
	StartAt string
	States  rawStates
}

// choiceKeys are the fields of a choice rule that are not its condition.
var choiceKeys = []string{"Next", "Variable", "Comment", "Assign", "Output"}

func describeChoice(choice rawChoice) string {
	var variable string
	_ = json.Unmarshal(choice["Variable"], &variable)

	var keys []string
	for key := range choice {
		if !slices.Contains(choiceKeys, key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	if len(keys) == 0 {
		return "when"
	}

	var key = keys[0]
	if key == "Condition" {
		var condition string
		_ = json.Unmarshal(choice[key], &condition)
		return "when " + condition
	}

	var value = string(choice[key])
	var compact bytes.Buffer
	if json.Compact(&compact, choice[key]) == nil {
		value = compact.String()
	}
	value = utils.ClampStringLen(&value, 40)

	if variable == "" {
		return fmt.Sprintf("when %s %s", key, value)
	}
	return fmt.Sprintf("when %s %s %s", variable, key, value)
}

func convertWorkflow(raw rawWorkflow) *aslWorkflow {
	var states = map[string]*aslState{}
	for _, name := range raw.States.names {
		states[name] = convertState(name, raw.States.states[name], raw.StartAt == name)
	}

	var workflow = &aslWorkflow{Comment: raw.Comment, StartAt: raw.StartAt}
	var visited = map[string]bool{}

	var visit func(name string)
	visit = func(name string) {
		var state, exists = states[name]
		if !exists || visited[name] {
			return
		}
		visited[name] = true
		workflow.States = append(workflow.States, state)
		for _, transition := range state.Transitions {
			visit(transition.Target)
		}
	}

	visit(raw.StartAt)
	for _, name := range raw.States.names {
		visit(name)
	}

	return workflow
}

func convertState(name string, raw rawState, start bool) *aslState {
	var state = &aslState{
		Name:    name,
		Type:    raw.Type,
		Comment: raw.Comment,
		Start:   start,
		End:     raw.End || raw.Type == "Succeed" || raw.Type == "Fail",
	}

	if raw.Next != "" {
		state.Transitions = append(state.Transitions, aslTransition{"next", raw.Next})
	}
	for _, choice := range raw.Choices {
		var next string
		_ = json.Unmarshal(choice["Next"], &next)
		if next != "" {
			state.Transitions = append(state.Transitions, aslTransition{describeChoice(choice), next})
		}
	}
	if raw.Default != "" {
		state.Transitions = append(state.Transitions, aslTransition{"default", raw.Default})
	}
	for _, catch := range raw.Catch {
		if catch.Next != "" {
			var label = "catch " + strings.Join(catch.ErrorEquals, ", ")
			state.Transitions = append(state.Transitions, aslTransition{label, catch.Next})
		}
	}

	for i, branch := range raw.Branches {
		state.Groups = append(state.Groups, aslGroup{
			Label:    fmt.Sprintf("Branch %d", i+1),
			Workflow: convertWorkflow(branch),
		})
	}
	if raw.ItemProcessor != nil {
		state.Groups = append(state.Groups, aslGroup{"Item processor", convertWorkflow(*raw.ItemProcessor)})
	}
	if raw.Iterator != nil {
		state.Groups = append(state.Groups, aslGroup{"Iterator", convertWorkflow(*raw.Iterator)})
	}

	return state
}

// parseWorkflow reads the Amazon States Language definition of a state machine.
func parseWorkflow(definition string) (*aslWorkflow, error) {
	var raw rawWorkflow
	if err := json.Unmarshal([]byte(definition), &raw); err != nil {
		return nil, err
	}
	if len(raw.States.names) == 0 {
		return nil, fmt.Errorf("the definition has no States")
	}

	return convertWorkflow(raw), nil
}
