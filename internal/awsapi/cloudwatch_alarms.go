package awsapi

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchAlarmsApi interface {
	ListAlarms(ctx context.Context, force bool) ([]types.MetricAlarm, error)
	ListAlarmHistory(ctx context.Context, name string, force bool) ([]types.AlarmHistoryItem, error)
}

type cloudWatchAlarmsApi struct {
	baseApi
	allCompositeAlarms map[string]types.CompositeAlarm
	alarmsPaginator    *cloudwatch.DescribeAlarmsPaginator
	historyPaginator   *cloudwatch.DescribeAlarmHistoryPaginator
}

func NewCloudWatchAlarmsApi(
	logger *log.Logger,
) CloudWatchAlarmsApi {
	return &cloudWatchAlarmsApi{
		baseApi:            baseApi{logger: logger},
		allCompositeAlarms: nil,
		alarmsPaginator:    nil,
		historyPaginator:   nil,
	}
}

func (inst *cloudWatchAlarmsApi) ListAlarms(ctx context.Context, force bool) ([]types.MetricAlarm, error) {
	inst.alarmsPaginator = cloudwatch.NewDescribeAlarmsPaginator(
		GetAwsApiClients().cloudwatch,
		&cloudwatch.DescribeAlarmsInput{
			MaxRecords: aws.Int32(100),
		},
	)

	var result, err = collectPaginator(inst.alarmsPaginator.HasMorePages, func() ([]types.MetricAlarm, error) {
		var output, err = inst.alarmsPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.MetricAlarms, nil
	})

	sortByKey(result, func(a types.MetricAlarm) *string { return a.AlarmName })

	return result, inst.logErr(err)
}

func (inst *cloudWatchAlarmsApi) ListAlarmHistory(ctx context.Context, name string, force bool) ([]types.AlarmHistoryItem, error) {
	if err := requireArg("alarm name", name); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().cloudwatch

	if force || inst.historyPaginator == nil {
		inst.historyPaginator = cloudwatch.NewDescribeAlarmHistoryPaginator(
			client,
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

	var output, err = inst.historyPaginator.NextPage(ctx)
	if err != nil {
		return foundHistory, inst.logErr(err)
	}

	foundHistory = append(foundHistory, output.AlarmHistoryItems...)

	return foundHistory, nil
}
