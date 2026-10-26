package cloudwatch

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func newAlarmDetailsTable() *AlarmDetailsTable {
	return NewAlarmDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchAlarmsApi]())
}

func TestAlarmDetailsTable__EmptyAlarmShowsBlankValues(t *testing.T) {
	var table = newAlarmDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"Description", ""},
		{"ARN", ""},
		{"State", ""},
		{"State Reason", ""},
		{"State Updated", ""},
		{"Metric"},
		{"Namespace", ""},
		{"Metric Name", ""},
		{"Statistic", ""},
		{"Period (s)", ""},
		{"Condition"},
		{"Comparison", ""},
		{"Threshold", ""},
		{"Datapoints to Alarm", ""},
		{"Evaluation Periods", ""},
	})
}

func TestAlarmDetailsTable__PopulatesFromAlarm(t *testing.T) {
	var updated = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var table = newAlarmDetailsTable()
	table.data = types.MetricAlarm{
		AlarmName:             aws.String("cpu-high"),
		AlarmDescription:      aws.String("CPU above 80%"),
		AlarmArn:              aws.String("arn:aws:cloudwatch:us-east-1:123456789012:alarm:cpu-high"),
		StateValue:            types.StateValueAlarm,
		StateReason:           aws.String("Threshold Crossed"),
		StateUpdatedTimestamp: &updated,
		Namespace:             aws.String("AWS/EC2"),
		MetricName:            aws.String("CPUUtilization"),
		Statistic:             types.StatisticAverage,
		Period:                aws.Int32(300),
		Dimensions:            []types.Dimension{{Name: aws.String("InstanceId"), Value: aws.String("i-123")}},
		ComparisonOperator:    types.ComparisonOperatorGreaterThanThreshold,
		Threshold:             aws.Float64(80),
		DatapointsToAlarm:     aws.Int32(2),
		EvaluationPeriods:     aws.Int32(3),
	}

	table.populateAlarmDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "cpu-high"},
		{"Description", "CPU above 80%"},
		{"ARN", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:cpu-high"},
		{"State", "ALARM"},
		{"State Reason", "Threshold Crossed"},
		{"State Updated", "2024-01-02 03:04:05"},
		{"Metric"},
		{"Namespace", "AWS/EC2"},
		{"Metric Name", "CPUUtilization"},
		{"Statistic", "Average"},
		{"Period (s)", "300"},
		{"InstanceId", "i-123"},
		{"Condition"},
		{"Comparison", "GreaterThanThreshold"},
		{"Threshold", "80"},
		{"Datapoints to Alarm", "2"},
		{"Evaluation Periods", "3"},
	})
	if got := table.GetTitle(); got != "Alarm Details ❬cpu-high❭" {
		t.Errorf("title = %q", got)
	}
}

func TestAlarmDetailsTable__PercentileAlarmShowsExtendedStatistic(t *testing.T) {
	var table = newAlarmDetailsTable()
	table.data = types.MetricAlarm{ExtendedStatistic: aws.String("p99")}

	table.populateAlarmDetailsTable()

	if got := table.GetCellText(9, 1); got != "p99" {
		t.Errorf("statistic = %q, want p99", got)
	}
}

func TestAlarmDetailsTable__RefreshReplacesPreviousAlarm(t *testing.T) {
	var serviceCtx = uitest.NewServiceContext[awsapi.CloudWatchAlarmsApi]()
	var table = NewAlarmDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshDetails(newAlarm("cpu-high", types.StateValueAlarm))
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetCellText(0, 1) == "cpu-high" })
	table.RefreshDetails(newAlarm("disk-ok", types.StateValueOk))

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return table.GetCellText(0, 1) == "disk-ok" && table.GetCellText(3, 1) == "OK"
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		if got := table.GetCellText(16, 0); got != "" {
			t.Errorf("unexpected row %q after the refresh", got)
		}
	})
}
