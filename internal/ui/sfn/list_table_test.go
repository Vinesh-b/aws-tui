package sfn

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

var stateMachineHeadings = core.TableRow{"Name", "Type", "Creation Date"}

func newPopulatedListTable(machines ...types.StateMachineListItem) *SfnListTable {
	var table = NewSfnListTable(uitest.NewServiceContext[awsapi.StateMachineApi]())
	table.populateTable(machines)
	return table
}

func TestListTable__ShowsHeadingsAndOneRowPerStateMachine(t *testing.T) {
	var table = newPopulatedListTable(
		newStateMachine("orders", types.StateMachineTypeStandard),
		newStateMachine("events", types.StateMachineTypeExpress),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		stateMachineHeadings,
		{"orders", "STANDARD", "2024-01-02 03:04:05"},
		{"events", "EXPRESS", "2024-01-02 03:04:05"},
	})
}

func TestListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{stateMachineHeadings})
}

func TestListTable__SelectingRowTracksSelectedStateMachine(t *testing.T) {
	var table = newPopulatedListTable(
		newStateMachine("orders", types.StateMachineTypeStandard),
		newStateMachine("events", types.StateMachineTypeExpress),
	)

	table.Select(2, 0)

	if got := table.GetSeletedFunctionName(); got != "events" {
		t.Errorf("name = %q, want %q", got, "events")
	}
	if got := table.GetSeletedFunctionArn(); got != "arn:aws:states:us-east-1:123456789012:stateMachine:events" {
		t.Errorf("arn = %q", got)
	}
	if got := table.GetSeletedFunctionType(); got != types.StateMachineTypeExpress {
		t.Errorf("type = %q, want %q", got, types.StateMachineTypeExpress)
	}
}
