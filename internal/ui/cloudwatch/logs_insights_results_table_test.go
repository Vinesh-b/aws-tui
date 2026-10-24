package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func resultRow(fields ...string) []types.ResultField {
	var row []types.ResultField
	for i := 0; i < len(fields); i += 2 {
		row = append(row, types.ResultField{Field: aws.String(fields[i]), Value: aws.String(fields[i+1])})
	}
	return row
}

func newResultsTable(rows ...[]types.ResultField) *InsightsQueryResultsTable {
	var table = NewInsightsQueryResultsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = rows
	table.populateQueryResultsTable()
	return table
}

func TestInsightsResultsTable__HeadingsFollowFieldOrder(t *testing.T) {
	var table = newResultsTable(
		resultRow("@timestamp", "2024-01-02", "@message", "hello"),
		resultRow("@timestamp", "2024-01-03", "@message", "world"),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"@timestamp", "@message"},
		{"2024-01-02", "hello"},
		{"2024-01-03", "world"},
	})
}

func TestInsightsResultsTable__PointerFieldIsHiddenButKeptForFirstColumn(t *testing.T) {
	var table = newResultsTable(
		resultRow("@timestamp", "2024-01-02", "@message", "hello", "@ptr", "ptr-1"),
		resultRow("@ptr", "ptr-2", "@timestamp", "2024-01-03", "@message", "world"),
	)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"@timestamp", "@message"},
		{"2024-01-02", "hello"},
		{"2024-01-03", "world"},
	})
	if got := table.GetRecordPtr(1); got != "ptr-1" {
		t.Errorf("row 1 pointer = %q, want %q", got, "ptr-1")
	}
	if got := table.GetRecordPtr(2); got != "ptr-2" {
		t.Errorf("row 2 pointer = %q, want %q", got, "ptr-2")
	}
}

func TestInsightsResultsTable__RowsMissingAFieldLeaveItBlank(t *testing.T) {
	var table = newResultsTable(
		resultRow("@timestamp", "t1", "@message", "hello"),
		resultRow("@timestamp", "t2"),
	)

	if got := table.GetCellText(2, 1); got != "" {
		t.Errorf("missing field = %q, want empty", got)
	}
}

func TestInsightsResultsTable__NoDataHasNoColumns(t *testing.T) {
	var table = newResultsTable()

	if got := table.GetColumnCount(); got != 0 {
		t.Errorf("column count = %d, want 0", got)
	}
}

func TestInsightsResultsTable__PrivateDataIsTheCellText(t *testing.T) {
	var table = newResultsTable(resultRow("@timestamp", "t1", "@message", "hello"))

	if got := table.GetPrivateData(1, 1); got != "hello" {
		t.Errorf("private data = %q, want %q", got, "hello")
	}
}

func TestInsightsQueryRunner__RequiresLogGroups(t *testing.T) {
	var runner = NewInsightsQueryRunner(uitest.NewAppContext().Ctx, uitest.NewAppContext().App, nil)
	var message string
	runner.ErrorMessageCallback = func(text string, a ...any) { message = text }

	runner.ExecuteInsightsQuery(InsightsQuery{}, nil, nil)

	if message != "No log groups selected" {
		t.Errorf("message = %q, want %q", message, "No log groups selected")
	}
}
