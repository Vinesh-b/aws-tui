package sqs

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newRunningQueueList(t *testing.T, api *fakeSqsApi) *QueueListTable {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SqsApi](api)
	var table = NewQueueListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func TestQueueListTable__OpenQueueLoadsListToSelectIt(t *testing.T) {
	var api = &fakeSqsApi{queues: []string{queueUrl("orders"), queueUrl("orders-dlq")}}
	var table = newRunningQueueList(t, api)

	uitest.OnUI(t, table.serviceCtx.App, func() { table.OpenQueue("orders-dlq") })

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return table.GetSelectedQueue() == queueUrl("orders-dlq")
	})
}

func TestQueueListTable__OpenMissingQueueReportsIt(t *testing.T) {
	var table = newRunningQueueList(t, &fakeSqsApi{queues: []string{queueUrl("orders")}})
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, table.serviceCtx.App, func() { table.OpenQueue("billing") })

	if got := uitest.AwaitError(t, received); got != "Queue billing not found" {
		t.Errorf("message = %q", got)
	}
}

func TestHomeView__OpensSqsResources(t *testing.T) {
	var page = NewSqsHomeView(uitest.NewAppContext(), &fakeSqsApi{})

	var opener, ok = page.(core.ResourceOpener)

	if !ok || !opener.OpensArnService("sqs") {
		t.Error("sqs page should open sqs resources")
	}
}

func findDetailsRow(table *core.DetailsTable, label string) int {
	const maxRows = 50
	for row := range maxRows {
		if table.GetCellText(row, 0) == label {
			return row
		}
	}
	return -1
}

func TestQueueDetailsTable__DeadLetterQueueLinksToQueue(t *testing.T) {
	var table = newDetailsTable()
	table.queueUrl = queueUrl("orders")
	table.attributes = map[string]string{
		"RedrivePolicy": `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:orders-dlq","maxReceiveCount":3}`,
	}

	table.populateQueueDetailsTable()

	var row = findDetailsRow(table.DetailsTable, "Dead-letter Queue")
	if row < 0 {
		t.Fatal("no dead-letter queue row")
	}
	if got := table.GetCellText(row, 1); got != "orders-dlq" {
		t.Errorf("value = %q, want orders-dlq", got)
	}
	if link, _ := table.GetLink(row); link != "arn:aws:sqs:us-east-1:123456789012:orders-dlq" {
		t.Errorf("link = %q", link)
	}
}

func TestQueueDetailsTable__NoDeadLetterQueueRowWithoutOne(t *testing.T) {
	var table = newDetailsTable()
	table.attributes = map[string]string{"RedrivePolicy": `{"maxReceiveCount":3}`}

	table.populateQueueDetailsTable()

	if row := findDetailsRow(table.DetailsTable, "Dead-letter Queue"); row >= 0 {
		t.Errorf("unexpected dead-letter queue row %d", row)
	}
}
