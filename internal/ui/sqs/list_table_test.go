package sqs

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

var queueHeadings = core.TableRow{"Name", "Type"}

func queueUrl(name string) string {
	return "https://sqs.us-east-1.amazonaws.com/123456789012/" + name
}

func newPopulatedQueueListTable(names ...string) *QueueListTable {
	var table = NewQueueListTable(uitest.NewServiceContext[awsapi.SqsApi]())
	var urls []string
	for _, name := range names {
		urls = append(urls, queueUrl(name))
	}
	table.populateQueueTable(urls)
	return table
}

func TestQueueListTable__ShowsHeadingsAndOneRowPerQueue(t *testing.T) {
	var table = newPopulatedQueueListTable("orders", "billing.fifo")

	uitest.AssertTableRows(t, table, []core.TableRow{
		queueHeadings,
		{"orders", "Standard"},
		{"billing.fifo", "FIFO"},
	})
}

func TestQueueListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedQueueListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{queueHeadings})
}

func TestQueueListTable__SelectingRowTracksSelectedQueueUrl(t *testing.T) {
	var table = newPopulatedQueueListTable("orders", "billing")

	table.Select(2, 0)

	if got := table.GetSelectedQueue(); got != queueUrl("billing") {
		t.Errorf("selected queue = %q, want %q", got, queueUrl("billing"))
	}
}

func TestQueueListTable__SelectedFuncReceivesRowAndColumn(t *testing.T) {
	var table = newPopulatedQueueListTable("orders")
	var gotRow, gotCol = -1, -1
	table.SetSelectedFunc(func(row, column int) { gotRow, gotCol = row, column })

	table.Select(1, 0)
	uitest.PressEnter(table)

	if gotRow != 1 || gotCol != 0 {
		t.Errorf("handler got (%d,%d), want (1,0)", gotRow, gotCol)
	}
}

func TestQueueListTable__SelectedFuncIgnoresHeadingRow(t *testing.T) {
	var table = newPopulatedQueueListTable("orders")
	var called = false
	table.SetSelectedFunc(func(row, column int) { called = true })

	table.Select(0, 0)
	uitest.PressEnter(table)

	if called {
		t.Error("handler was called for the heading row")
	}
}
