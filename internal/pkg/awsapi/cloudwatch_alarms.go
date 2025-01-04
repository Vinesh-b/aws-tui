package awsapi

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchAlarmsApi struct {
	logger             *log.Logger
	config             aws.Config
	client             *cloudwatch.Client
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
		allCompositeAlarms: nil,
		alarmsPaginator:    nil,
		historyPaginator:   nil,
	}
}

func (inst *CloudWatchAlarmsApi) ListAlarms(force bool) ([]types.MetricAlarm, error) {
	inst.alarmsPaginator = cloudwatch.NewDescribeAlarmsPaginator(
		inst.client,
		&cloudwatch.DescribeAlarmsInput{
			MaxRecords: aws.Int32(100),
		},
	)

	var apiErr error = nil
	var result = []types.MetricAlarm{}

	for inst.alarmsPaginator.HasMorePages() {
		var output, err = inst.alarmsPaginator.NextPage(context.TODO())
		if err != nil {
			apiErr = err
			break
		}

		result = append(result, output.MetricAlarms...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].AlarmName) < aws.ToString(result[j].AlarmName)
	})

	return result, apiErr
}

func (inst *CloudWatchAlarmsApi) ListAlarmHistory(name string, force bool) ([]types.AlarmHistoryItem, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("Alarm name not set")
	}

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
		return foundHistory, nil
	}

	var output, err = inst.historyPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return foundHistory, err
	}

	foundHistory = append(foundHistory, output.AlarmHistoryItems...)

	return foundHistory, nil
}
