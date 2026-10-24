package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func newPopulatedListTable(names ...string) *LambdaListTable {
	var table = NewLambdasListTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	var functions []types.FunctionConfiguration
	for _, name := range names {
		functions = append(functions, newFunction(name))
	}
	table.populateLambdasTable(functions)
	return table
}

func TestListTable__ShowsHeadingsAndOneRowPerFunction(t *testing.T) {
	var table = newPopulatedListTable("alpha", "beta")

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "LastModified"},
		{"alpha", "2024-01-02T03:04:05.000+0000"},
		{"beta", "2024-01-02T03:04:05.000+0000"},
	})
}

func TestListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name", "LastModified"}})
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
	uitest.PressEnter(table)

	if gotRow != 1 || gotCol != 0 {
		t.Errorf("handler got (%d,%d), want (1,0)", gotRow, gotCol)
	}
}

func TestListTable__SelectedFuncIgnoresHeadingRow(t *testing.T) {
	var table = newPopulatedListTable("alpha")
	var called = false
	table.SetSelectedFunc(func(row, column int) { called = true })

	table.Select(0, 0)
	uitest.PressEnter(table)

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
			var table = NewLambdasListTable(uitest.NewServiceContext[awsapi.LambdaApi]())
			table.populateLambdasTable([]types.FunctionConfiguration{tt.function})

			if got := table.GetSeletedLambdaLogGroup(); got != tt.want {
				t.Errorf("log group = %q, want %q", got, tt.want)
			}
		})
	}
}
