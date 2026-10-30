package cloudformation

import (
	"context"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type fakeCloudFormationApi struct {
	awsapi.CloudFormationApi
	stacks []types.StackSummary
	events []types.StackEvent
	err    error
}

func (f *fakeCloudFormationApi) ListStacks(ctx context.Context, force bool) ([]types.StackSummary, error) {
	return f.stacks, f.err
}

func (f *fakeCloudFormationApi) DescribeStackEvents(
	ctx context.Context, stackName string, force bool,
) ([]types.StackEvent, error) {
	return f.events, f.err
}

func TestStackListTable__RefreshLoadsStacksFromApi(t *testing.T) {
	var api = &fakeCloudFormationApi{stacks: []types.StackSummary{newStack("orders", types.StackStatusCreateComplete)}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudFormationApi](api)
	var table = NewStackListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshStacks(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		stackListHeadings,
		{"orders", "CREATE_COMPLETE", "2024-02-03 04:05:06"},
	})
}

func TestStackListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeCloudFormationApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudFormationApi](api)
	var table = NewStackListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshStacks(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestStackListTable__SearchShowsMatchingStacks(t *testing.T) {
	var api = &fakeCloudFormationApi{stacks: []types.StackSummary{
		newStack("orders", types.StackStatusCreateComplete),
		newStack("billing", types.StackStatusCreateComplete),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudFormationApi](api)
	var table = NewStackListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshStacks(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("bill") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		stackListHeadings,
		{"billing", "CREATE_COMPLETE", "2024-02-03 04:05:06"},
	})
}

func TestStackEventsTable__RefreshLoadsEventsForSelectedStack(t *testing.T) {
	var api = &fakeCloudFormationApi{events: []types.StackEvent{
		newStackEvent("Bucket", types.ResourceStatusCreateComplete, ""),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudFormationApi](api)
	var table = NewStackEventsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedStackName("orders")

	table.RefreshEvents(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		stackEventHeadings,
		{"2024-01-02 03:04:05.678", "Bucket", "AWS::S3::Bucket", "CREATE_COMPLETE", ""},
	})
}

func TestStackEventsTable__RefreshWithoutSelectedStackShowsNothing(t *testing.T) {
	var api = &fakeCloudFormationApi{events: []types.StackEvent{
		newStackEvent("Bucket", types.ResourceStatusCreateComplete, ""),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudFormationApi](api)
	var table = NewStackEventsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshEvents(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{stackEventHeadings})
}
