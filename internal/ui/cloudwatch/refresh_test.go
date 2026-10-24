package cloudwatch

import (
	"context"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type fakeCloudWatchAlarmsApi struct {
	awsapi.CloudWatchAlarmsApi
	alarms  []cwTypes.MetricAlarm
	history []cwTypes.AlarmHistoryItem
	err     error
}

func (f *fakeCloudWatchAlarmsApi) ListAlarms(ctx context.Context, force bool) ([]cwTypes.MetricAlarm, error) {
	return f.alarms, f.err
}

func (f *fakeCloudWatchAlarmsApi) ListAlarmHistory(
	ctx context.Context, name string, force bool,
) ([]cwTypes.AlarmHistoryItem, error) {
	return f.history, f.err
}

type fakeCloudWatchMetricsApi struct {
	awsapi.CloudWatchMetricsApi
	metrics []cwTypes.Metric
	err     error
}

func (f *fakeCloudWatchMetricsApi) ListMetrics(
	ctx context.Context, dims []cwTypes.DimensionFilter, namespace string, metricName string, force bool,
) ([]cwTypes.Metric, error) {
	return f.metrics, f.err
}

type fakeCloudWatchLogsApi struct {
	awsapi.CloudWatchLogsApi
	groups  []types.LogGroup
	streams []types.LogStream
	events  []types.OutputLogEvent
	err     error
}

func (f *fakeCloudWatchLogsApi) ListLogGroups(ctx context.Context, reset bool) ([]types.LogGroup, error) {
	return f.groups, f.err
}

func (f *fakeCloudWatchLogsApi) ListLogStreams(
	ctx context.Context, logGroupName string, searchPrefix string, reset bool,
) ([]types.LogStream, error) {
	return f.streams, f.err
}

func (f *fakeCloudWatchLogsApi) ListLogEvents(
	ctx context.Context, logGroupName string, logStreamName string, reset bool,
) ([]types.OutputLogEvent, error) {
	return f.events, f.err
}

func TestAlarmListTable__RefreshLoadsAlarmsFromApi(t *testing.T) {
	var api = &fakeCloudWatchAlarmsApi{alarms: []cwTypes.MetricAlarm{newAlarm("cpu-high", cwTypes.StateValueAlarm)}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshAlarms(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name", "State"}, {"cpu-high", "ALARM"}})
}

func TestAlarmListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeCloudWatchAlarmsApi{err: errors.New("access denied")}
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

func TestAlarmListTable__FilterByNameShowsMatchingAlarms(t *testing.T) {
	var api = &fakeCloudWatchAlarmsApi{alarms: []cwTypes.MetricAlarm{
		newAlarm("cpu-high", cwTypes.StateValueAlarm), newAlarm("disk-ok", cwTypes.StateValueOk),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchAlarmsApi](api)
	var table = NewAlarmListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshAlarms(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterbyName("disk")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name", "State"}, {"disk-ok", "OK"}})
}

func TestAlarmHistoryTable__RefreshLoadsHistoryFromApi(t *testing.T) {
	var api = &fakeCloudWatchAlarmsApi{history: []cwTypes.AlarmHistoryItem{
		newHistoryItem("Alarm updated from OK to ALARM", 0),
	}}
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
	var api = &fakeCloudWatchMetricsApi{metrics: []cwTypes.Metric{newMetric("AWS/EC2", "CPUUtilization")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var table = NewMetricsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshMetrics(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Namespace", "Name"},
		{"AWS/EC2", "CPUUtilization"},
	})
}

func TestMetricListTable__FilterByNameShowsMatchingMetrics(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{metrics: []cwTypes.Metric{
		newMetric("AWS/EC2", "CPUUtilization"), newMetric("AWS/S3", "BucketSizeBytes"),
	}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var table = NewMetricsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshMetrics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("bucket")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Namespace", "Name"},
		{"AWS/S3", "BucketSizeBytes"},
	})
}

func TestLogGroupsTable__RefreshLoadsGroupsFromApi(t *testing.T) {
	var api = &fakeCloudWatchLogsApi{groups: []types.LogGroup{newLogGroup("/aws/lambda/orders")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshLogGroups(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name"}, {"/aws/lambda/orders"}})
}

func TestLogGroupsTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeCloudWatchLogsApi{err: errors.New("access denied")}
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

func TestLogGroupsTable__FilterByNameShowsMatchingGroups(t *testing.T) {
	var api = &fakeCloudWatchLogsApi{groups: []types.LogGroup{newLogGroup("/aws/lambda/orders"), newLogGroup("/aws/ecs/web")}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshLogGroups(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("ecs")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{{"Name"}, {"/aws/ecs/web"}})
}

func TestLogStreamsTable__RefreshLoadsStreamsFromApi(t *testing.T) {
	var api = &fakeCloudWatchLogsApi{streams: []types.LogStream{newLogStream("stream-a", 0)}}
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
	var api = &fakeCloudWatchLogsApi{events: []types.OutputLogEvent{newLogEvent("START", 0)}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchLogsApi](api)
	var table = NewLogEventsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshLogEvents(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		{"Timestamp", "Message"},
		{"2024-01-02 03:04:05.678", "START"},
	})
}
