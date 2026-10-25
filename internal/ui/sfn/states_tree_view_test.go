package sfn

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func nodeTexts(node *tview.TreeNode) []string {
	var texts []string
	node.Walk(func(n, _ *tview.TreeNode) bool {
		texts = append(texts, n.GetText())
		return true
	})
	return texts
}

func describedWith(definition string) *sfn.DescribeStateMachineOutput {
	var machine = newStateMachineDescription()
	machine.Definition = aws.String(definition)
	return machine
}

func TestStatesTreeView__ShowsAHintUntilAStateMachineIsSelected(t *testing.T) {
	var view = NewSfnStatesTreeView()

	if got := view.GetRoot().GetText(); got != noStatesMessage {
		t.Errorf("root = %q, want the hint", got)
	}
}

func TestStatesTreeView__ShowsStatesInFlowOrderWithTheirTransitions(t *testing.T) {
	var view = NewSfnStatesTreeView()

	view.SetStateMachine(describedWith(orderDefinition))

	var all = strings.Join(nodeTexts(view.GetRoot()), "\n")
	for _, want := range []string{
		"orders  orders",
		"Validate [Pass] (start)",
		"→ next: Route",
		"Route [Choice]",
		"→ default: Failed",
		"Failed [Fail] (end)",
		"Unused [Pass] (end)",
	} {
		if !strings.Contains(all, tview.Escape(want)) {
			t.Errorf("tree is missing %q:\n%s", want, all)
		}
	}
	if got := view.GetCurrentNode().GetText(); !strings.HasPrefix(got, "Validate") {
		t.Errorf("selected node = %q, want the start state", got)
	}
}

func TestStatesTreeView__EnterOnATransitionJumpsToItsState(t *testing.T) {
	var view = NewSfnStatesTreeView()
	view.SetStateMachine(describedWith(orderDefinition))

	var validate = view.GetRoot().GetChildren()[0]
	view.nodeSelected(validate.GetChildren()[0])

	if got := view.GetCurrentNode().GetText(); !strings.HasPrefix(got, "Route") {
		t.Errorf("selected node = %q, want Route", got)
	}
}

func TestStatesTreeView__ShowsNestedWorkflowsBelowTheirState(t *testing.T) {
	var view = NewSfnStatesTreeView()

	view.SetStateMachine(describedWith(`{"StartAt": "Fan", "States": {
	  "Fan": {"Type": "Parallel", "End": true, "Branches": [
	    {"StartAt": "A", "States": {"A": {"Type": "Pass", "End": true}}}
	  ]}
	}}`))

	var fan = view.GetRoot().GetChildren()[0]
	var group = fan.GetChildren()[0]
	if group.GetText() != "Branch 1" {
		t.Fatalf("group = %q, want Branch 1", group.GetText())
	}
	if got := group.GetChildren()[0].GetText(); !strings.HasPrefix(got, tview.Escape("A [Pass]")) {
		t.Errorf("branch state = %q", got)
	}
}

func TestStatesTreeView__ExplainsADefinitionItCannotShow(t *testing.T) {
	var view = NewSfnStatesTreeView()

	view.SetStateMachine(describedWith("not json"))

	if got := view.GetRoot().GetText(); !strings.HasPrefix(got, "Cannot show the states") {
		t.Errorf("root = %q, want an explanation", got)
	}
}

func stateNode(t *testing.T, view *SfnStatesTreeView, name string) *tview.TreeNode {
	t.Helper()
	var node, ok = view.stateNodes[name]
	if !ok {
		t.Fatalf("no node for state %q", name)
	}
	return node
}

func newExecutionTree(t *testing.T) *SfnStatesTreeView {
	t.Helper()
	var view = NewSfnStatesTreeView()
	view.SetStateMachine(describedWith(orderDefinition))
	view.SetExecution([]StateDetails{
		stateWith("Validate", SfnStateTypePass, 1, "PassStateEntered", "PassStateExited"),
		stateWith("Route", SfnStateTypeChoice, 0, "ChoiceStateEntered", "ChoiceStateExited"),
		stateWith("Charge", SfnStateTypeTask, 2, "TaskStateEntered", "TaskFailed"),
	})
	return view
}

func TestStatesTreeView__ExecutionColoursStatesByOutcome(t *testing.T) {
	var view = newExecutionTree(t)

	for name, want := range map[string]tcell.Color{
		"Validate": tcell.ColorGreen,
		"Charge":   tcell.ColorRed,
		"Done":     tcell.ColorGrey,
		"Unused":   tcell.ColorGrey,
	} {
		if got := stateNode(t, view, name).GetColor(); got != want {
			t.Errorf("%s colour = %v, want %v", name, got, want)
		}
	}
}

func TestStatesTreeView__ExecutionShowsDurationOnReachedStatesOnly(t *testing.T) {
	var view = newExecutionTree(t)

	if got := stateNode(t, view, "Validate").GetText(); !strings.HasSuffix(got, "✔ 1s") {
		t.Errorf("validate text = %q, want it to end with the outcome and duration", got)
	}
	if got := stateNode(t, view, "Done").GetText(); strings.Contains(got, "✔") || strings.Contains(got, "✘") {
		t.Errorf("done was not reached but shows %q", got)
	}
}

func TestStatesTreeView__NoExecutionStatesRestoresTheTree(t *testing.T) {
	var view = newExecutionTree(t)

	view.SetExecution(nil)

	if got := stateNode(t, view, "Failed").GetColor(); got != tcell.ColorRed {
		t.Errorf("failed colour = %v, want its own red", got)
	}
	if got := stateNode(t, view, "Validate").GetText(); strings.Contains(got, "✔") {
		t.Errorf("validate still shows an outcome: %q", got)
	}
	if got := stateNode(t, view, "Done").GetColor(); got != tcell.ColorGreen {
		t.Errorf("done colour = %v, want the succeed green", got)
	}
}

func TestStatesTreeView__NewStateMachineDropsTheExecution(t *testing.T) {
	var view = newExecutionTree(t)

	view.SetStateMachine(describedWith(orderDefinition))

	if got := stateNode(t, view, "Validate").GetText(); strings.Contains(got, "✔") {
		t.Errorf("validate still shows the old execution: %q", got)
	}
}

func TestStatesTreeView__SelectStateMovesTheCursor(t *testing.T) {
	var view = newExecutionTree(t)

	view.SelectState("Charge")
	if got := view.GetCurrentNode().GetText(); !strings.HasPrefix(got, "Charge") {
		t.Errorf("cursor on %q, want Charge", got)
	}

	view.SelectState("Nope")
	if got := view.GetCurrentNode().GetText(); !strings.HasPrefix(got, "Charge") {
		t.Errorf("cursor moved to %q for an unknown state", got)
	}
}
