package sqs

import (
	"context"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

type fakeSqsApi struct {
	awsapi.SqsApi
	queues     []string
	attributes map[string]string
	tags       map[string]string
	err        error
}

func (f *fakeSqsApi) ListQueues(ctx context.Context, force bool) ([]string, error) {
	return f.queues, f.err
}

func (f *fakeSqsApi) GetQueueAttributes(ctx context.Context, queueUrl string) (map[string]string, error) {
	return f.attributes, f.err
}

func (f *fakeSqsApi) ListQueueTags(ctx context.Context, queueUrl string) (map[string]string, error) {
	return f.tags, f.err
}

func TestQueueListTable__RefreshLoadsQueuesFromApi(t *testing.T) {
	var api = &fakeSqsApi{queues: []string{queueUrl("orders")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshQueues(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		queueHeadings,
		{"orders", "Standard"},
	})
}

func TestQueueListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeSqsApi{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshQueues(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestQueueListTable__SearchShowsMatchingQueues(t *testing.T) {
	var api = &fakeSqsApi{queues: []string{queueUrl("orders"), queueUrl("billing")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshQueues(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("bill") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		queueHeadings,
		{"billing", "Standard"},
	})
}

func TestQueueDetailsTable__RefreshLoadsAttributesFromApi(t *testing.T) {
	var api = &fakeSqsApi{attributes: newQueueAttributes()}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshDetails(queueUrl("orders"))

	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(8, 1) == "5" })
}

func TestQueueDetailsTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeSqsApi{err: errors.New("queue does not exist")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshDetails(queueUrl("orders"))

	if got := uitest.AwaitError(t, received); got != "queue does not exist" {
		t.Errorf("message = %q, want %q", got, "queue does not exist")
	}
}
