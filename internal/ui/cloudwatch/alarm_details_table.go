package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type AlarmDetailsTable struct {
	*core.DetailsTable
	data       types.MetricAlarm
	serviceCtx *core.ServiceContext[awsapi.CloudWatchAlarmsApi]
}

func NewAlarmDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchAlarmsApi],
) *AlarmDetailsTable {
	var view = &AlarmDetailsTable{
		DetailsTable: core.NewDetailsTable("Alarm Details", serviceContext.AppContext),
		data:         types.MetricAlarm{},
		serviceCtx:   serviceContext,
	}

	view.populateAlarmDetailsTable()
	return view
}

// statistic is the standard statistic of the alarm, or the percentile it uses
func (inst *AlarmDetailsTable) statistic() string {
	if len(inst.data.Statistic) > 0 {
		return string(inst.data.Statistic)
	}
	return aws.ToString(inst.data.ExtendedStatistic)
}

// dimensionLink is the ARN of the resource the dimension names, for the
// services that can be opened. Alarms are in the account and region of what
// they watch.
func (inst *AlarmDetailsTable) dimensionLink(name string, value string) string {
	if len(value) == 0 {
		return ""
	}

	var alarmArn = aws.ToString(inst.data.AlarmArn)
	switch dimension := aws.ToString(inst.data.Namespace) + "/" + name; dimension {
	case "AWS/Lambda/FunctionName":
		return core.ArnFor(alarmArn, "lambda", "function:"+value)
	case "AWS/SQS/QueueName":
		return core.ArnFor(alarmArn, "sqs", value)
	}
	return ""
}

func (inst *AlarmDetailsTable) populateAlarmDetailsTable() {
	var data = inst.data
	var tableData = []core.TableRow{
		{"Name", aws.ToString(data.AlarmName)},
		{"Description", aws.ToString(data.AlarmDescription)},
		{"ARN", aws.ToString(data.AlarmArn)},
		{"State", string(data.StateValue)},
		{"State Reason", aws.ToString(data.StateReason)},
		{"State Updated", core.FormatTime(data.StateUpdatedTimestamp)},
	}

	var metric = []core.TableRow{
		{"Namespace", aws.ToString(data.Namespace)},
		{"Metric Name", aws.ToString(data.MetricName)},
		{"Statistic", inst.statistic()},
		{"Period (s)", core.FormatOptionalInt(data.Period)},
	}
	for _, dim := range data.Dimensions {
		var name, value = aws.ToString(dim.Name), aws.ToString(dim.Value)
		metric = append(metric, core.DetailsLink(name, value, inst.dimensionLink(name, value)))
	}
	tableData = append(tableData, core.DetailsSection("Metric", metric...)...)

	tableData = append(tableData, core.DetailsSection("Condition",
		core.TableRow{"Comparison", string(data.ComparisonOperator)},
		core.TableRow{"Threshold", core.FormatOptionalFloat(data.Threshold)},
		core.TableRow{"Datapoints to Alarm", core.FormatOptionalInt(data.DatapointsToAlarm)},
		core.TableRow{"Evaluation Periods", core.FormatOptionalInt(data.EvaluationPeriods)},
	)...)

	inst.SetTitleExtra(aws.ToString(data.AlarmName))
	inst.SetRawData(inst.data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *AlarmDetailsTable) RefreshDetails(alarm types.MetricAlarm) {
	inst.data = alarm
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateAlarmDetailsTable()
	})
}
