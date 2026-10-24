package cloudformation

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

var stackListHeadings = core.TableRow{"StackName", "Status", "LastUpdated"}

func newPopulatedStackListTable(stacks ...types.StackSummary) *StackListTable {
	var table = NewStackListTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())
	table.populateStacksTable(stacks)
	return table
}

func TestStackListTable__ShowsHeadingsAndOneRowPerStack(t *testing.T) {
	var neverUpdated = newStack("legacy", types.StackStatusCreateComplete)
	neverUpdated.LastUpdatedTime = nil

	var table = newPopulatedStackListTable(
		newStack("orders", types.StackStatusUpdateComplete),
		neverUpdated,
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		stackListHeadings,
		{"orders", "UPDATE_COMPLETE", "2024-02-03 04:05:06"},
		{"legacy", "CREATE_COMPLETE", "-"},
	})
}

func TestStackListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedStackListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{stackListHeadings})
}

func TestStackListTable__SelectingRowTracksSelectedStack(t *testing.T) {
	var table = newPopulatedStackListTable(
		newStack("orders", types.StackStatusUpdateComplete),
		newStack("billing", types.StackStatusCreateComplete),
	)

	table.Select(2, 0)

	if got := table.GetSelectedStackName(); got != "billing" {
		t.Errorf("selected stack = %q, want %q", got, "billing")
	}
}

func TestStackListTable__SelectingHeadingKeepsPreviousSelection(t *testing.T) {
	var table = newPopulatedStackListTable(
		newStack("orders", types.StackStatusUpdateComplete),
		newStack("billing", types.StackStatusCreateComplete),
	)
	table.Select(2, 0)

	table.Select(0, 0)

	if got := table.GetSelectedStackName(); got != "billing" {
		t.Errorf("selected stack = %q, want %q", got, "billing")
	}
}
