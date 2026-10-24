package eventbridge

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

var eventBusHeadings = core.TableRow{"Name", "Description", "LastModified"}

func newPopulatedEventBusListTable(names ...string) *EventBusListTable {
	var table = NewEventBusListTable(uitest.NewServiceContext[awsapi.EventBridgeApi]())
	var buses []types.EventBus
	for _, name := range names {
		buses = append(buses, newEventBus(name))
	}
	table.populateEventBusTable(buses)
	return table
}

func TestEventBusListTable__ShowsHeadingsAndOneRowPerBus(t *testing.T) {
	var table = newPopulatedEventBusListTable("orders", "billing")

	uitest.AssertTableRows(t, table, []core.TableRow{
		eventBusHeadings,
		{"orders", "orders bus", "2024-01-02 03:04:05"},
		{"billing", "billing bus", "2024-01-02 03:04:05"},
	})
}

func TestEventBusListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedEventBusListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{eventBusHeadings})
}

func TestEventBusListTable__SelectingRowTracksSelectedBus(t *testing.T) {
	var table = newPopulatedEventBusListTable("orders", "billing")

	table.Select(2, 0)

	if got := aws.ToString(table.GetSeletedEventBus().Name); got != "billing" {
		t.Errorf("selected bus = %q, want %q", got, "billing")
	}
}

func TestEventBusListTable__SelectedFuncReceivesRowAndColumn(t *testing.T) {
	var table = newPopulatedEventBusListTable("orders")
	var gotRow, gotCol = -1, -1
	table.SetSelectedFunc(func(row, column int) { gotRow, gotCol = row, column })

	table.Select(1, 0)
	uitest.PressEnter(table)

	if gotRow != 1 || gotCol != 0 {
		t.Errorf("handler got (%d,%d), want (1,0)", gotRow, gotCol)
	}
}

func TestEventBusListTable__SelectedFuncIgnoresHeadingRow(t *testing.T) {
	var table = newPopulatedEventBusListTable("orders")
	var called = false
	table.SetSelectedFunc(func(row, column int) { called = true })

	table.Select(0, 0)
	uitest.PressEnter(table)

	if called {
		t.Error("handler was called for the heading row")
	}
}
