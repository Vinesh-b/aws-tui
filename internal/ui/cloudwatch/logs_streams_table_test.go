package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func newLogStream(name string, lastEventSeconds int) types.LogStream {
	return types.LogStream{
		LogStreamName:      aws.String(name),
		LastEventTimestamp: millisAt(lastEventSeconds),
	}
}

func newPopulatedLogStreamsTable(streams ...types.LogStream) *LogStreamsTable {
	var table = NewLogStreamsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = streams
	table.populateLogStreamsTable(false)
	return table
}

func TestLogStreamsTable__ShowsHeadingsAndOneRowPerStream(t *testing.T) {
	var table = newPopulatedLogStreamsTable(newLogStream("stream-a", 0), newLogStream("stream-b", 60))

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Name", "LastEventTimestamp"},
		{"stream-a", "2024-01-02 03:04:05"},
		{"stream-b", "2024-01-02 03:05:05"},
	})
}

func TestLogStreamsTable__SelectedFuncTracksSelectedStream(t *testing.T) {
	var table = newPopulatedLogStreamsTable(newLogStream("stream-a", 0), newLogStream("stream-b", 60))
	table.SetSelectedFunc(func(row, column int) {})

	table.Select(2, 0)
	uitest.PressEnter(table)

	if got := table.GetSeletedLogStream(); got != "stream-b" {
		t.Errorf("selected stream = %q, want %q", got, "stream-b")
	}
}

func TestLogStreamsTable__LogStreamDetailForSelectedStream(t *testing.T) {
	var table = newPopulatedLogStreamsTable(newLogStream("stream-a", 0), newLogStream("stream-b", 60))
	table.SetSelectionChangedFunc(func(row, column int) {})

	table.Select(2, 0)

	if got := aws.ToString(table.GetLogStreamDetail().LogStreamName); got != "stream-b" {
		t.Errorf("detail stream = %q, want %q", got, "stream-b")
	}
}

func TestLogStreamsTable__LogStreamDetailIsEmptyForUnknownStream(t *testing.T) {
	var table = newPopulatedLogStreamsTable(newLogStream("stream-a", 0))

	if detail := table.GetLogStreamDetail(); detail.LogStreamName != nil {
		t.Errorf("detail = %+v, want empty", detail)
	}
}

func TestLogStreamsTable__SettingLogGroupIsRemembered(t *testing.T) {
	var table = newPopulatedLogStreamsTable()

	table.SetSeletedLogGroup("/aws/lambda/orders")

	if got := table.GetSeletedLogGroup(); got != "/aws/lambda/orders" {
		t.Errorf("log group = %q, want %q", got, "/aws/lambda/orders")
	}
}
