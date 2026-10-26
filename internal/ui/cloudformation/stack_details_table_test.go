package cloudformation

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestStackDetailsTable__EmptyStackShowsPlaceholders(t *testing.T) {
	var table = NewStackDetailsTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"Stack ID", ""},
		{"Description", ""},
		{"Status", ""},
		{"Status Reason", ""},
		{"Created", ""},
		{"Last Updated", ""},
	})
}

func TestStackDetailsTable__PopulatesFromStack(t *testing.T) {
	var table = NewStackDetailsTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())
	table.data = newStack("orders", types.StackStatusUpdateComplete)
	table.data.StackId = aws.String("stack-id-1")
	table.data.TemplateDescription = aws.String("Orders stack")
	table.data.StackStatusReason = aws.String("user initiated")

	table.populateStackDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "orders"},
		{"Stack ID", "stack-id-1"},
		{"Description", "Orders stack"},
		{"Status", "UPDATE_COMPLETE"},
		{"Status Reason", "user initiated"},
		{"Created", "2024-01-02 03:04:05"},
		{"Last Updated", "2024-02-03 04:05:06"},
	})
}

func newStack(name string, status types.StackStatus) types.StackSummary {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var updated = time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	return types.StackSummary{
		StackName:       aws.String(name),
		StackStatus:     status,
		CreationTime:    &created,
		LastUpdatedTime: &updated,
	}
}
