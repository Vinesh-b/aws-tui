package dynamodb

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newPopulatedTablesTable(names ...string) *DynamoDBTablesTable {
	var table = NewDynamoDBTablesTable(uitest.NewServiceContext[awsapi.DynamoDBApi]())
	table.data = names
	table.populateTablesTable()
	return table
}

func TestTablesTable__ShowsHeadingAndOneRowPerTable(t *testing.T) {
	var table = newPopulatedTablesTable("orders", "users")

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}, {"orders"}, {"users"}})
}

func TestTablesTable__EmptyDataShowsOnlyHeading(t *testing.T) {
	var table = newPopulatedTablesTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}})
}

func TestTablesTable__SelectingRowTracksSelectedTable(t *testing.T) {
	var table = newPopulatedTablesTable("orders", "users")

	table.Select(2, 0)

	if got := table.GetSelectedTable(); got != "users" {
		t.Errorf("selected table = %q, want %q", got, "users")
	}
}

func TestTablesTable__SelectedFuncIgnoresHeadingRow(t *testing.T) {
	var table = newPopulatedTablesTable("orders")
	var called = false
	table.SetSelectedFunc(func(row, column int) { called = true })

	table.Select(0, 0)
	uitest.PressEnter(table)

	if called {
		t.Error("handler was called for the heading row")
	}
}
