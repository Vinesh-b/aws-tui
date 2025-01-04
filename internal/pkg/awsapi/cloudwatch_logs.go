package awsapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudWatchLogsApi struct {
	logger                     *log.Logger
	config                     aws.Config
	client                     *cloudwatchlogs.Client
	logEventsPaginator         *cloudwatchlogs.GetLogEventsPaginator
	logStreamsPaginator        *cloudwatchlogs.DescribeLogStreamsPaginator
	logGroupsPaginator         *cloudwatchlogs.DescribeLogGroupsPaginator
	filteredLogEventsPaginator *cloudwatchlogs.FilterLogEventsPaginator
}

func NewCloudWatchLogsApi(
	config aws.Config,
	logger *log.Logger,
) *CloudWatchLogsApi {
	return &CloudWatchLogsApi{
		config: config,
		logger: logger,
		client: cloudwatchlogs.NewFromConfig(config),
	}
}

func (inst *CloudWatchLogsApi) ListLogGroups(reset bool) ([]types.LogGroup, error) {
	if reset || inst.logGroupsPaginator == nil {
		inst.logGroupsPaginator = cloudwatchlogs.NewDescribeLogGroupsPaginator(
			inst.client,
			&cloudwatchlogs.DescribeLogGroupsInput{
				Limit: aws.Int32(50),
			},
		)
	}

	var apiErr error = nil
	var result = []types.LogGroup{}

	for inst.logGroupsPaginator.HasMorePages() {
		var output, err = inst.logGroupsPaginator.NextPage(context.TODO())
		if err != nil {
			apiErr = err
			break
		}

		result = append(result, output.LogGroups...)
	}

	sort.Slice(result, func(i, j int) bool {
		return aws.ToString(result[i].LogGroupName) < aws.ToString(result[j].LogGroupName)
	})

	return result, apiErr
}

func (inst *CloudWatchLogsApi) ListLogStreams(
	logGroupName string,
	searchPrefix string,
	reset bool,
) ([]types.LogStream, error) {
	var empty = []types.LogStream{}

	if len(logGroupName) == 0 {
		return empty, fmt.Errorf("log group not set")
	}

	var searchPrefixPtr *string = nil

	if reset || inst.logStreamsPaginator == nil {
		var order = types.OrderByLastEventTime
		if len(searchPrefix) == 0 {
			searchPrefixPtr = nil
		} else {
			order = types.OrderByLogStreamName
			searchPrefixPtr = &searchPrefix
		}
		inst.logStreamsPaginator = cloudwatchlogs.NewDescribeLogStreamsPaginator(
			inst.client,
			&cloudwatchlogs.DescribeLogStreamsInput{
				Descending:          aws.Bool(true),
				Limit:               aws.Int32(50),
				LogGroupName:        aws.String(logGroupName),
				LogStreamNamePrefix: searchPrefixPtr,
				OrderBy:             order,
			},
		)
	}

	if !inst.logStreamsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.logStreamsPaginator.NextPage(context.TODO())

	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.LogStreams, nil
}

func (inst *CloudWatchLogsApi) ListLogEvents(
	logGroupName string,
	logStreamName string,
	reset bool,
) ([]types.OutputLogEvent, error) {
	var empty = []types.OutputLogEvent{}

	if len(logGroupName) == 0 {
		return empty, fmt.Errorf("log group not set")
	}

	if len(logStreamName) == 0 {
		return empty, fmt.Errorf("log stream not set")
	}

	if reset || inst.logEventsPaginator == nil {
		inst.logEventsPaginator = cloudwatchlogs.NewGetLogEventsPaginator(
			inst.client,
			&cloudwatchlogs.GetLogEventsInput{
				LogStreamName: aws.String(logStreamName),
				Limit:         aws.Int32(500),
				LogGroupName:  aws.String(logGroupName),
				StartFromHead: aws.Bool(true),
			},
		)
	}

	if !inst.logEventsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.logEventsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.Events, nil
}

func (inst *CloudWatchLogsApi) ListFilteredLogEvents(
	logGroupName string,
	startDateTime time.Time,
	endDateTime time.Time,
	reset bool,
) ([]types.FilteredLogEvent, error) {
	var empty = []types.FilteredLogEvent{}

	if len(logGroupName) == 0 {
		return empty, fmt.Errorf("log group not set")
	}

	if reset || inst.filteredLogEventsPaginator == nil {
		inst.filteredLogEventsPaginator = cloudwatchlogs.NewFilterLogEventsPaginator(
			inst.client,
			&cloudwatchlogs.FilterLogEventsInput{
				Limit:        aws.Int32(500),
				LogGroupName: aws.String(logGroupName),
				StartTime:    aws.Int64(startDateTime.UnixMilli()),
				EndTime:      aws.Int64(endDateTime.UnixMilli()),
			},
		)
	}
	if !inst.filteredLogEventsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.filteredLogEventsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.Events, nil
}

func (inst *CloudWatchLogsApi) StartInightsQuery(
	logGroups []string,
	startTime time.Time,
	endTime time.Time,
	query string,
) (string, error) {
	var output, err = inst.client.StartQuery(
		context.TODO(), &cloudwatchlogs.StartQueryInput{
			StartTime:     aws.Int64(startTime.Unix()),
			EndTime:       aws.Int64(endTime.Unix()),
			LogGroupNames: logGroups,
			QueryString:   aws.String(query),
		},
	)

	if err != nil {
		inst.logger.Println(err)
		return "", err
	}

	return aws.ToString(output.QueryId), nil
}

func (inst *CloudWatchLogsApi) StopInightsQuery(
	queryId string,
) (bool, error) {
	if len(queryId) == 0 {
		return false, fmt.Errorf("Query Id not set")
	}

	var output, err = inst.client.StopQuery(
		context.TODO(), &cloudwatchlogs.StopQueryInput{
			QueryId: aws.String(queryId),
		},
	)

	if err != nil {
		inst.logger.Println(err)
		return false, err
	}

	return *aws.Bool(output.Success), nil
}

func (inst *CloudWatchLogsApi) GetInightsQueryResults(
	queryId string,
) ([][]types.ResultField, types.QueryStatus, error) {
	var empty [][]types.ResultField

	if len(queryId) == 0 {
		return empty, types.QueryStatusUnknown, fmt.Errorf("Query Id not set")
	}

	var output, err = inst.client.GetQueryResults(
		context.TODO(), &cloudwatchlogs.GetQueryResultsInput{
			QueryId: aws.String(queryId),
		})

	if err != nil {
		inst.logger.Println(err)
		return empty, types.QueryStatusUnknown, err
	}

	return output.Results, output.Status, nil
}

func (inst *CloudWatchLogsApi) GetInsightsLogRecord(
	recordPtr string,
) (map[string]string, error) {
	var empty = map[string]string{}

	if len(recordPtr) == 0 {
		return empty, fmt.Errorf("Record pointer not set")
	}

	var output, err = inst.client.GetLogRecord(
		context.TODO(), &cloudwatchlogs.GetLogRecordInput{
			LogRecordPointer: aws.String(recordPtr),
		})

	if err != nil {
		inst.logger.Println(err)
		return empty, err
	}

	return output.LogRecord, nil
}
