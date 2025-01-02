package awsapi

import (
	"aws-tui/internal/pkg/ui/core"
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
	allMetricAlarms    []types.MetricAlarm
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
		allMetricAlarms:    []types.MetricAlarm{},
		allCompositeAlarms: nil,
		alarmsPaginator:    nil,
		historyPaginator:   nil,
	}
}

func (inst *CloudWatchAlarmsApi) ListAlarms(force bool) ([]types.MetricAlarm, error) {
	if len(inst.allMetricAlarms) > 0 && !force {
		return inst.allMetricAlarms, nil
	}

	inst.allMetricAlarms = []types.MetricAlarm{}
	inst.alarmsPaginator = cloudwatch.NewDescribeAlarmsPaginator(
		inst.client,
		&cloudwatch.DescribeAlarmsInput{
			MaxRecords: aws.Int32(100),
		},
	)

	var err error = nil
	var output *cloudwatch.DescribeAlarmsOutput
	for inst.alarmsPaginator.HasMorePages() {
		output, err = inst.alarmsPaginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Println(err)
			break
		}

		inst.allMetricAlarms = append(inst.allMetricAlarms, output.MetricAlarms...)
	}

	sort.Slice(inst.allMetricAlarms, func(i, j int) bool {
		return aws.ToString(inst.allMetricAlarms[i].AlarmName) < aws.ToString(inst.allMetricAlarms[j].AlarmName)
	})

	return inst.allMetricAlarms, err
}

func (inst *CloudWatchAlarmsApi) FilterByName(name string) []types.MetricAlarm {
	if len(inst.allMetricAlarms) == 0 {
		return nil
	}

	var foundIdxs = core.FuzzySearch(name, inst.allMetricAlarms, func(a types.MetricAlarm) string {
		return aws.ToString(a.AlarmName)
	})

	var foundAlarms []types.MetricAlarm

	for _, matchIdx := range foundIdxs {
		var alarm = inst.allMetricAlarms[matchIdx]
		foundAlarms = append(foundAlarms, alarm)
	}
	return foundAlarms
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
