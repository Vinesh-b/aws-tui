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

var stackEventHeadings = core.TableRow{"Timestamp", "LogicalId", "ResourceType", "Status", "Reason"}

func newStackEvent(logicalId string, status types.ResourceStatus, reason string) types.StackEvent {
	var timestamp = time.Date(2024, 1, 2, 3, 4, 5, 678000000, time.UTC)
	return types.StackEvent{
		Timestamp:            &timestamp,
		LogicalResourceId:    aws.String(logicalId),
		ResourceType:         aws.String("AWS::S3::Bucket"),
		ResourceStatus:       status,
		ResourceStatusReason: aws.String(reason),
	}
}

func TestStackEventsTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = NewStackEventsTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())

	uitest.AssertTableRows(t, table, []core.TableRow{stackEventHeadings})
}

func TestStackEventsTable__ShowsOneRowPerEvent(t *testing.T) {
	var table = NewStackEventsTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())
	table.data = []types.StackEvent{
		newStackEvent("Bucket", types.ResourceStatusCreateComplete, ""),
		newStackEvent("Queue", types.ResourceStatusCreateFailed, "access denied"),
	}

	table.populateStackEventsTable(true)

	uitest.AssertTableRows(t, table, []core.TableRow{
		stackEventHeadings,
		{"2024-01-02 03:04:05.678", "Bucket", "AWS::S3::Bucket", "CREATE_COMPLETE", ""},
		{"2024-01-02 03:04:05.678", "Queue", "AWS::S3::Bucket", "CREATE_FAILED", "access denied"},
	})
}

func TestStackEventsTable__ResourceStatusReasonForRow(t *testing.T) {
	var table = NewStackEventsTable(uitest.NewServiceContext[awsapi.CloudFormationApi]())
	table.data = []types.StackEvent{
		newStackEvent("Bucket", types.ResourceStatusCreateComplete, ""),
		newStackEvent("Queue", types.ResourceStatusCreateFailed, "access denied"),
	}
	table.populateStackEventsTable(true)

	if got := table.GetResourceStatusReason(2); got != "access denied" {
		t.Errorf("reason = %q, want %q", got, "access denied")
	}
}
