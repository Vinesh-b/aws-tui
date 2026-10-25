package dynamodb

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"
)

func newResultsTable(data ...map[string]any) *DynamoDBGenericTable {
	var table = NewDynamoDBGenericTable(uitest.NewServiceContext[awsapi.DynamoDBApi]())
	table.data = data
	return table
}

func headings(table *DynamoDBGenericTable) []string {
	var result []string
	for col := 0; table.GetCellText(0, col) != ""; col++ {
		result = append(result, table.GetCellText(0, col))
	}
	return result
}

func cellFor(table *DynamoDBGenericTable, row int, heading string) string {
	var col = slices.Index(headings(table), heading)
	return table.GetCellText(row, col)
}

func TestGenericTable__WithoutTableDescriptionNothingIsShown(t *testing.T) {
	var table = newResultsTable(map[string]any{"id": "u1"})

	table.populateDynamoDBTable(false)

	if got := table.GetTable().GetRowCount(); got != 0 {
		t.Errorf("row count = %d, want 0", got)
	}
}

func TestGenericTable__KeyColumnsComeFirst(t *testing.T) {
	var table = newResultsTable(map[string]any{"name": "alice", "ts": "2024", "id": "u1", "age": 30})
	table.tableDescription = newTableDescription("users", hashKey("id"), rangeKey("ts"))

	table.populateDynamoDBTable(false)

	var got = headings(table)
	if len(got) != 4 || got[0] != "id" || got[1] != "ts" {
		t.Errorf("headings = %v, want id and ts first of 4 columns", got)
	}
	if table.pkName != "id" || table.skName != "ts" {
		t.Errorf("pk, sk = %q, %q, want id, ts", table.pkName, table.skName)
	}
}

func TestGenericTable__EachRowShowsItsAttributeValues(t *testing.T) {
	var table = newResultsTable(
		map[string]any{"id": "u1", "name": "alice", "age": 30},
		map[string]any{"id": "u2", "name": "bob"},
	)
	table.tableDescription = newTableDescription("users", hashKey("id"))

	table.populateDynamoDBTable(false)

	var tests = []struct {
		row     int
		heading string
		want    string
	}{
		{1, "id", "u1"},
		{1, "name", "alice"},
		{1, "age", "30"},
		{2, "id", "u2"},
		{2, "name", "bob"},
		{2, "age", "<nil>"},
	}
	for _, tt := range tests {
		if got := cellFor(table, tt.row, tt.heading); got != tt.want {
			t.Errorf("row %d %q = %q, want %q", tt.row, tt.heading, got, tt.want)
		}
	}
}

func TestGenericTable__FirstCellHoldsTheFullRow(t *testing.T) {
	var table = newResultsTable(map[string]any{"id": "u1", "name": "alice"})
	table.tableDescription = newTableDescription("users", hashKey("id"))

	table.populateDynamoDBTable(false)

	var row = table.GetPrivateData(1, 0)
	if row["id"] != "u1" || row["name"] != "alice" {
		t.Errorf("private data = %v, want the full item", row)
	}
}

func TestGenericTable__SelectingRowRemembersIt(t *testing.T) {
	var table = newResultsTable(map[string]any{"id": "u1"}, map[string]any{"id": "u2"})
	table.tableDescription = newTableDescription("users", hashKey("id"))
	table.populateDynamoDBTable(false)

	table.Select(2, 0)

	if table.lastSelectedRowIdx != 2 {
		t.Errorf("last selected row = %d, want 2", table.lastSelectedRowIdx)
	}
}

func TestGenericTable__SelectedItemHasAllAttributes(t *testing.T) {
	var table = newResultsTable(
		map[string]any{"id": "u1", "name": "alice"},
		map[string]any{"id": "u2", "name": "bob"},
	)
	table.tableDescription = newTableDescription("users", hashKey("id"))
	table.populateDynamoDBTable(false)

	table.GetTable().Select(2, 1)
	var item, ok = table.GetSelectedItem()

	if !ok || item["id"] != "u2" || item["name"] != "bob" {
		t.Errorf("GetSelectedItem = %v, %v, want the second item", item, ok)
	}
}

func TestGenericTable__NoSelectedItemWhenOnlyHeadingsOrEmpty(t *testing.T) {
	var table = newResultsTable()
	if _, ok := table.GetSelectedItem(); ok {
		t.Error("expected no item from an empty table")
	}

	table = newResultsTable(map[string]any{"id": "u1"})
	table.tableDescription = newTableDescription("users", hashKey("id"))
	table.populateDynamoDBTable(false)
	table.GetTable().Select(0, 0)
	if _, ok := table.GetSelectedItem(); ok {
		t.Error("expected no item for the heading row")
	}
}
