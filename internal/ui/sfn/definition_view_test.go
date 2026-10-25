package sfn

import (
	"strings"
	"testing"

	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

func newDefinitionView() *SfnDefinitionView {
	return NewSfnDefinitionView(uitest.NewAppContext())
}

func TestDefinitionView__ShowsAHintUntilAStateMachineIsSelected(t *testing.T) {
	var view = newDefinitionView()

	if got := view.GetText(); got != noDefinitionMessage {
		t.Errorf("text = %q, want the hint", got)
	}
}

func TestDefinitionView__IndentsAJsonDefinition(t *testing.T) {
	var view = newDefinitionView()
	var machine = newStateMachineDescription()
	machine.Definition = aws.String(`{"StartAt":"A","States":{"A":{"Type":"Pass","End":true}}}`)

	view.SetDefinition(machine)

	var text = view.GetText()
	if !strings.Contains(text, "\n  \"StartAt\": \"A\",") {
		t.Errorf("definition was not indented:\n%s", text)
	}
	if !view.JsonHighlighting() {
		t.Error("a json definition should be highlighted")
	}
}

func TestDefinitionView__ShowsADefinitionThatIsNotJsonAsIs(t *testing.T) {
	var view = newDefinitionView()
	var machine = newStateMachineDescription()
	machine.Definition = aws.String("not json")

	view.SetDefinition(machine)

	if got := view.GetText(); got != "not json" {
		t.Errorf("text = %q, want the definition unchanged", got)
	}
	if view.JsonHighlighting() {
		t.Error("text that is not json should not be highlighted")
	}
}

func TestDefinitionView__ClearingTheStateMachineShowsTheHintAgain(t *testing.T) {
	var view = newDefinitionView()
	var machine = newStateMachineDescription()
	machine.Definition = aws.String(`{"StartAt":"A"}`)
	view.SetDefinition(machine)

	view.SetDefinition(nil)

	if got := view.GetText(); got != noDefinitionMessage {
		t.Errorf("text = %q, want the hint", got)
	}
}

func TestDetailsTable__ReportsTheDescribedStateMachineWhenShown(t *testing.T) {
	var table = newDetailsTable()
	var got = "never called"
	table.SetDetailsLoadedFunc(func(machine *sfn.DescribeStateMachineOutput) {
		got = aws.ToString(machine.Name)
	})
	table.data = newStateMachineDescription()

	table.populateTable()

	if got != "orders" {
		t.Errorf("loaded func saw %q, want orders", got)
	}
}
