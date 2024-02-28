package cloudwatch

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
}

func NewCloudWatchAlarmsApi(
	config aws.Config,
	logger *log.Logger,
) *CloudWatchAlarmsApi {
	return &CloudWatchAlarmsApi{
		config: config,
		logger: logger,
		client: cloudwatch.NewFromConfig(config),
        allMetricAlarms: make(map[string]types.MetricAlarm),
	}
}

func (inst *CloudWatchAlarmsApi) ListAlarms(force bool) map[string]types.MetricAlarm {
	if len(inst.allMetricAlarms) > 0 && !force {
		return inst.allMetricAlarms
	}

	var nextToken *string = nil

	for {
		output, err := inst.client.DescribeAlarms(
			context.TODO(), &cloudwatch.DescribeAlarmsInput{
				MaxRecords: aws.Int32(50),
				NextToken:  nextToken,
			},
		)

		if err != nil {
			inst.logger.Println(err)
			break
		}

		nextToken = output.NextToken

		for _, val := range output.MetricAlarms {
			inst.allMetricAlarms[*val.AlarmName] = val
		}

		if nextToken == nil {
			break
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

func (inst *CloudWatchAlarmsApi) ListAlarmHistory(name string) []types.AlarmHistoryItem {
	var nextToken *string = nil
	var foundHistory []types.AlarmHistoryItem

	for {
		output, err := inst.client.DescribeAlarmHistory(
			context.TODO(), &cloudwatch.DescribeAlarmHistoryInput{
				AlarmName:  aws.String(name),
				MaxRecords: aws.Int32(50),
				NextToken:  nextToken,
			},
		)

		if err != nil {
			inst.logger.Println(err)
			break
		}

		nextToken = output.NextToken
		foundHistory = append(foundHistory, output.AlarmHistoryItems...)
		if nextToken == nil {
			break
		}
	}

	return foundHistory
}
