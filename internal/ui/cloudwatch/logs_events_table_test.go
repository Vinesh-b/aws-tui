package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func newLogEvent(message string, seconds int) types.OutputLogEvent {
	return types.OutputLogEvent{Message: aws.String(message), Timestamp: millisAt(seconds)}
}

func TestLogEventsTable__ShowsHeadingsAndOneRowPerEvent(t *testing.T) {
	var table = NewLogEventsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = []types.OutputLogEvent{newLogEvent("START", 0), newLogEvent("END", 1)}

	table.populateLogEventsTable(true)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Timestamp", "Message"},
		{"2024-01-02 03:04:05.678", "START"},
		{"2024-01-02 03:04:06.678", "END"},
	})
}

func TestLogEventsTable__FullLogMessageForRow(t *testing.T) {
	var table = NewLogEventsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = []types.OutputLogEvent{newLogEvent("START", 0), newLogEvent("END", 1)}
	table.populateLogEventsTable(true)

	var tests = []struct {
		row  int
		want string
	}{
		{0, ""},
		{1, "START"},
		{2, "END"},
		{3, ""},
	}
	for _, tt := range tests {
		if got := table.GetFullLogMessage(tt.row); got != tt.want {
			t.Errorf("row %d message = %q, want %q", tt.row, got, tt.want)
		}
	}
}
