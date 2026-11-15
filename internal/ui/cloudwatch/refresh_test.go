package cloudwatch

import (
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestAlarmListTable__RefreshLoadsAlarmsFromApi(t *testing.T) {
	var api = cloudwatchfake.NewAlarms(cloudwatchfake.WithAlarms([]cwTypes.MetricAlarm{newAlarm("cpu-high", cwTypes.StateValueAlarm)}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshAlarms(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name", "State"}, {"cpu-high", "ALARM"}})
}

func TestAlarmListTable__RefreshReportsApiError(t *testing.T) {
	var api = cloudwatchfake.NewAlarms(cloudwatchfake.WithAlarmsErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshAlarms(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestAlarmListTable__SearchShowsMatchingAlarms(t *testing.T) {
	var api = cloudwatchfake.NewAlarms(
		cloudwatchfake.WithAlarms([]cwTypes.MetricAlarm{
			newAlarm("cpu-high", cwTypes.StateValueAlarm), newAlarm("disk-ok", cwTypes.StateValueOk),
		}),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshAlarms(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("disk") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name", "State"}, {"disk-ok", "OK"}})
}

func TestAlarmHistoryTable__RefreshLoadsHistoryFromApi(t *testing.T) {
	var api = cloudwatchfake.NewAlarms(
		cloudwatchfake.WithAlarmHistory([]cwTypes.AlarmHistoryItem{
			newHistoryItem("Alarm updated from OK to ALARM", 0),
		}),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmHistoryTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedAlarm("cpu-high")

	table.RefreshHistory(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "History"},
		{"2024-01-02 03:04:05", "Alarm updated from OK to ALARM"},
	})
}

func TestMetricListTable__RefreshLoadsMetricsFromApi(t *testing.T) {
	var api = cloudwatchfake.NewMetrics(cloudwatchfake.WithMetrics([]cwTypes.Metric{newMetric("AWS/EC2", "CPUUtilization")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var table = NewMetricsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshMetrics(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "Namespace"},
		{"CPUUtilization", "AWS/EC2"},
	})
}

func TestMetricListTable__SearchShowsMatchingMetrics(t *testing.T) {
	var api = cloudwatchfake.NewMetrics(
		cloudwatchfake.WithMetrics([]cwTypes.Metric{
			newMetric("AWS/EC2", "CPUUtilization"), newMetric("AWS/S3", "BucketSizeBytes"),
		}),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var table = NewMetricsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshMetrics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("bucket") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "Namespace"},
		{"BucketSizeBytes", "AWS/S3"},
	})
}

func TestMetricListTable__SearchMatchesDimensionValues(t *testing.T) {
	var api = cloudwatchfake.NewMetrics(
		cloudwatchfake.WithMetrics([]cwTypes.Metric{
			newMetric("AWS/Lambda", "Invocations", cwTypes.Dimension{Name: aws.String("FunctionName"), Value: aws.String("orders")}),
			newMetric("AWS/Lambda", "Invocations", cwTypes.Dimension{Name: aws.String("FunctionName"), Value: aws.String("billing")}),
		}),
	)
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var table = NewMetricsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshMetrics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("billing") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "Namespace"},
		{"Invocations (FunctionName=billing)", "AWS/Lambda"},
	})
}

func TestLogGroupsTable__RefreshLoadsGroupsFromApi(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogGroups([]types.LogGroup{newLogGroup("/aws/lambda/orders")}...))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshLogGroups(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"", "Name"}, {"", "/aws/lambda/orders"}})
}

func TestLogGroupsTable__RefreshReportsApiError(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshLogGroups(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestLogGroupsTable__SearchFiltersToTheMatchingGroups(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogGroups([]types.LogGroup{newLogGroup("/aws/lambda/orders"), newLogGroup("/aws/ecs/web")}...))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshLogGroups(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("ecs") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"", "Name"}, {"", "/aws/ecs/web"}})
}

func TestLogStreamsTable__RefreshLoadsStreamsFromApi(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogStreams([]types.LogStream{newLogStream("stream-a", 0)}...))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogStreamsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedLogGroup("/aws/lambda/orders")

	table.RefreshStreams(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Name", "LastEventTimestamp"},
		{"stream-a", "2024-01-02 03:04:05"},
	})
}

func TestLogEventsTable__RefreshLoadsEventsFromApi(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogEvents([]types.OutputLogEvent{newLogEvent("START", 0)}...))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogEventsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshLogEvents(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "Message"},
		{"2024-01-02 03:04:05.678", "START"},
	})
}

func TestLogStreamsTable__MovingDownFromTheLastRowLoadsTheNextPage(t *testing.T) {
	var api = cloudwatchfake.New(cloudwatchfake.WithLogStreams([]types.LogStream{newLogStream("stream-a", 0), newLogStream("stream-b", 0)}...))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogStreamsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshStreams(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 }, "the first page")

	uitest.OnUI(t, serviceCtx.App, func() {
		table.Select(2, 0)
		uitest.PressTableRune(table, 'j')
	})

	// The fake gives the same streams for every page
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 5 }, "the next page")
	uitest.OnUI(t, serviceCtx.App, func() {
		if row, _ := table.GetTable().GetSelection(); row != 2 {
			t.Errorf("row = %d, want the cursor to stay on 2", row)
		}
	})
}
