package lambda

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newPopulatedListTable(names ...string) *LambdaListTable {
	var table = NewLambdasListTable(newTestServiceContext())
	var functions []types.FunctionConfiguration
	for _, name := range names {
		functions = append(functions, newFunction(name))
	}
	table.populateLambdasTable(functions)
	return table
}

func pressEnter(table *LambdaListTable) {
	var handler = table.GetTable().InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}

func TestListTable__ShowsHeadingsAndOneRowPerFunction(t *testing.T) {
	var table = newPopulatedListTable("alpha", "beta")

	var want = [][]string{
		{"Name", "LastModified"},
		{"alpha", "2024-01-02T03:04:05.000+0000"},
		{"beta", "2024-01-02T03:04:05.000+0000"},
	}
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if got := table.GetTable().GetRowCount(); got != len(want) {
		t.Errorf("row count = %d, want %d", got, len(want))
	}
}

func TestListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedListTable()

	if got := table.GetTable().GetRowCount(); got != 1 {
		t.Errorf("row count = %d, want 1", got)
	}
}

func TestListTable__SelectingRowTracksSelectedLambda(t *testing.T) {
	var table = newPopulatedListTable("alpha", "beta")

	table.Select(2, 0)

	if got := aws.ToString(table.GetSeletedLambda().FunctionName); got != "beta" {
		t.Errorf("selected lambda = %q, want %q", got, "beta")
	}
}

func TestListTable__SelectingHeadingKeepsPreviousSelection(t *testing.T) {
	var table = newPopulatedListTable("alpha", "beta")
	table.Select(2, 0)

	table.Select(0, 0)

	if got := aws.ToString(table.GetSeletedLambda().FunctionName); got != "beta" {
		t.Errorf("selected lambda = %q, want %q", got, "beta")
	}
}

func TestListTable__SelectedFuncReceivesRowAndColumn(t *testing.T) {
	var table = newPopulatedListTable("alpha", "beta")
	var gotRow, gotCol = -1, -1
	table.SetSelectedFunc(func(row, column int) { gotRow, gotCol = row, column })

	table.Select(1, 0)
	pressEnter(table)

	if gotRow != 1 || gotCol != 0 {
		t.Errorf("handler got (%d,%d), want (1,0)", gotRow, gotCol)
	}
}

func TestListTable__SelectedFuncIgnoresHeadingRow(t *testing.T) {
	var table = newPopulatedListTable("alpha")
	var called = false
	table.SetSelectedFunc(func(row, column int) { called = true })

	table.Select(0, 0)
	pressEnter(table)

	if called {
		t.Error("handler was called for the heading row")
	}
}

func TestListTable__SelectedLambdaLogGroup(t *testing.T) {
	var withLogGroup = newFunction("alpha")
	withLogGroup.LoggingConfig = &types.LoggingConfig{LogGroup: aws.String("/aws/lambda/alpha")}
	var tests = []struct {
		name     string
		function types.FunctionConfiguration
		want     string
	}{
		{"configured log group", withLogGroup, "/aws/lambda/alpha"},
		{"no logging config", newFunction("beta"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var table = NewLambdasListTable(newTestServiceContext())
			table.populateLambdasTable([]types.FunctionConfiguration{tt.function})

			if got := table.GetSeletedLambdaLogGroup(); got != tt.want {
				t.Errorf("log group = %q, want %q", got, tt.want)
			}
		})
	}
}
