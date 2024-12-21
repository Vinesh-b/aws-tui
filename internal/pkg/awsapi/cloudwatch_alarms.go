package awsapi

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchAlarmsApi struct {
	logger             *log.Logger
	config             aws.Config
	client             *cloudwatch.Client
	allMetricAlarms    map[string]types.MetricAlarm
	allCompositeAlarms map[string]types.CompositeAlarm
	alarmsPaginator    *cloudwatch.DescribeAlarmsPaginator
	historyPaginator   *cloudwatch.DescribeAlarmHistoryPaginator
}

func NewCloudWatchAlarmsApi(
	config aws.Config,
	logger *log.Logger,
) *CloudWatchAlarmsApi {
	return &CloudWatchAlarmsApi{
		config:             config,
		logger:             logger,
		client:             cloudwatch.NewFromConfig(config),
		allMetricAlarms:    make(map[string]types.MetricAlarm),
		allCompositeAlarms: nil,
		alarmsPaginator:    nil,
		historyPaginator:   nil,
	}
}

func (inst *CloudWatchAlarmsApi) ListAlarms(force bool) map[string]types.MetricAlarm {
	if len(inst.allMetricAlarms) > 0 && !force {
		return inst.allMetricAlarms
	}

	inst.allMetricAlarms = make(map[string]types.MetricAlarm)
	inst.alarmsPaginator = cloudwatch.NewDescribeAlarmsPaginator(
		inst.client,
		&cloudwatch.DescribeAlarmsInput{
			MaxRecords: aws.Int32(100),
		},
	)

	for inst.alarmsPaginator.HasMorePages() {
		var output, err = inst.alarmsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		for _, val := range output.MetricAlarms {
			inst.allMetricAlarms[*val.AlarmName] = val
		}
	}

	return inst.allMetricAlarms
}

func (inst *CloudWatchAlarmsApi) FilterByName(name string) map[string]types.MetricAlarm {

	if len(inst.allMetricAlarms) < 1 {
		inst.ListAlarms(true)
	}

	var foundAlarms = make(map[string]types.MetricAlarm)

	for _, info := range inst.allMetricAlarms {
		found := strings.Contains(*info.AlarmName, name)
		if found {
			foundAlarms[*info.AlarmName] = info
		}
	}
	return foundAlarms
}

func (inst *CloudWatchAlarmsApi) ListAlarmHistory(name string, force bool) []types.AlarmHistoryItem {
	if force || inst.historyPaginator == nil {
		inst.historyPaginator = cloudwatch.NewDescribeAlarmHistoryPaginator(
			inst.client,
			&cloudwatch.DescribeAlarmHistoryInput{
				AlarmName:  aws.String(name),
				MaxRecords: aws.Int32(50),
			},
		)
	}

	var foundHistory []types.AlarmHistoryItem
	if !inst.historyPaginator.HasMorePages() {
		return foundHistory
	}

	var output, err = inst.historyPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return foundHistory
	}

	foundHistory = append(foundHistory, output.AlarmHistoryItems...)

	return foundHistory
}
