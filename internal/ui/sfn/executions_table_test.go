package sfn

import (
	"slices"
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/gdamore/tcell/v2"
)

var executionHeadings = core.TableRow{"Execution Id", "Status", "Start Date", "Stop Date", "Duration"}

func newExecutionsTable() *SfnExecutionsTable {
	return NewSfnExecutionsTable(uitest.NewAppContext(), nil, nil)
}

func executionNames(items []ExecutionItem) []string {
	var names []string
	for _, item := range items {
		names = append(names, aws.ToString(item.Name))
	}
	return names
}

func TestExecutionsTable__ShowsHeadingsAndOneRowPerExecution(t *testing.T) {
	var table = newExecutionsTable()
	table.data = []ExecutionItem{
		newExecution("run-1", types.ExecutionStatusSucceeded, 0, 90),
		newExecution("run-2", types.ExecutionStatusFailed, 10, 15),
	}

	table.populateTable(true)

	uitest.AssertTableRows(t, table, []core.TableRow{
		executionHeadings,
		{"run-1", "SUCCEEDED", "2024-01-02 03:04:05", "2024-01-02 03:05:35", "1m30s"},
		{"run-2", "FAILED", "2024-01-02 03:04:15", "2024-01-02 03:04:20", "5s"},
	})
}

func TestExecutionsTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newExecutionsTable()

	table.populateTable(true)

	uitest.AssertTableRows(t, table, []core.TableRow{executionHeadings})
}

func TestExecutionsTable__SelectedFuncTracksSelectedExecution(t *testing.T) {
	var table = newExecutionsTable()
	table.data = []ExecutionItem{
		newExecution("run-1", types.ExecutionStatusSucceeded, 0, 1),
		newExecution("run-2", types.ExecutionStatusFailed, 0, 1),
	}
	table.populateTable(true)
	table.SetSelectedFunc(func(row, column int) {})

	table.Select(2, 0)
	uitest.PressEnter(table)

	if got := aws.ToString(table.selectedExecution.Name); got != "run-2" {
		t.Errorf("selected execution = %q, want %q", got, "run-2")
	}
}

func TestExecutionsTable__FilterByStatus(t *testing.T) {
	var table = newExecutionsTable()
	var data = []ExecutionItem{
		newExecution("run-1", types.ExecutionStatusSucceeded, 0, 1),
		newExecution("run-2", types.ExecutionStatusFailed, 0, 1),
		newExecution("run-3", types.ExecutionStatusSucceeded, 0, 1),
	}

	var tests = []struct {
		status string
		want   []string
	}{
		{"ALL", []string{"run-1", "run-2", "run-3"}},
		{"SUCCEEDED", []string{"run-1", "run-3"}},
		{"FAILED", []string{"run-2"}},
		{"ABORTED", nil},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			var got = executionNames(table.FilterByStatus(data, tt.status))

			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionsTable__FilterByExecutionId(t *testing.T) {
	var table = newExecutionsTable()
	var data = []ExecutionItem{
		newExecution("run-1", types.ExecutionStatusSucceeded, 0, 1),
		newExecution("run-2", types.ExecutionStatusFailed, 0, 1),
	}

	var tests = []struct {
		name string
		id   string
		want []string
	}{
		{"empty id keeps everything", "", []string{"run-1", "run-2"}},
		{"matching id", "run-2", []string{"run-2"}},
		{"unknown id", "run-9", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got = executionNames(table.FilterByExecutionId(data, tt.id))

			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionsTable__ColoursSucceededAndFailedStatuses(t *testing.T) {
	var table = newExecutionsTable()
	table.data = []ExecutionItem{
		newExecution("run-1", types.ExecutionStatusSucceeded, 0, 1),
		newExecution("run-2", types.ExecutionStatusFailed, 0, 1),
		newExecution("run-3", types.ExecutionStatusRunning, 0, 1),
	}

	table.populateTable(true)

	var uncoloured, _, _ = table.GetCell(3, 0).Style.Decompose()
	var tests = []struct {
		row  int
		want tcell.Color
	}{
		{1, tcell.ColorForestGreen},
		{2, tcell.ColorIndianRed},
		{3, uncoloured},
	}
	for _, tt := range tests {
		var fg, _, _ = table.GetCell(tt.row, 1).Style.Decompose()
		if fg != tt.want {
			t.Errorf("row %d colour = %v, want %v", tt.row, fg, tt.want)
		}
	}
}
