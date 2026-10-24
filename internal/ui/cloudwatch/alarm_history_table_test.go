package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func newHistoryItem(summary string, seconds int) types.AlarmHistoryItem {
	return types.AlarmHistoryItem{
		HistorySummary: aws.String(summary),
		Timestamp:      aws.Time(baseTime.Add(0)),
	}
}

func TestAlarmHistoryTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = NewAlarmHistoryTable(uitest.NewServiceContext[awsapi.CloudWatchAlarmsApi]())

	uitest.AssertTableRows(t, table, []core.TableRow{{"Timestamp", "History"}})
}

func TestAlarmHistoryTable__ShowsOneRowPerHistoryItem(t *testing.T) {
	var table = NewAlarmHistoryTable(uitest.NewServiceContext[awsapi.CloudWatchAlarmsApi]())
	table.data = []types.AlarmHistoryItem{
		newHistoryItem("Alarm updated from OK to ALARM", 0),
		newHistoryItem("Alarm updated from ALARM to OK", 0),
	}

	table.populateAlarmHistoryTable(true)

	uitest.AssertTableRows(t, table, []core.TableRow{
		{"Timestamp", "History"},
		{"2024-01-02 03:04:05", "Alarm updated from OK to ALARM"},
		{"2024-01-02 03:04:05", "Alarm updated from ALARM to OK"},
	})
}
