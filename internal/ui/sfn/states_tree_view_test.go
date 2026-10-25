package sfn

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
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
