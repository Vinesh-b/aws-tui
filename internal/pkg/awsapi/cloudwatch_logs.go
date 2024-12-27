package awsapi

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudWatchLogsApi struct {
	logger              *log.Logger
	config              aws.Config
	client              *cloudwatchlogs.Client
	allLogGroups        []types.LogGroup
	logEventsPaginator  *cloudwatchlogs.GetLogEventsPaginator
	logStreamsPaginator *cloudwatchlogs.DescribeLogStreamsPaginator
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

func (inst *CloudWatchLogsApi) ListLogGroups(force bool) []types.LogGroup {
	if len(inst.allLogGroups) > 0 && !force {
		return inst.allLogGroups
	}

	inst.allLogGroups = nil
	var nextToken *string = nil

	for {
		output, err := inst.client.DescribeLogGroups(
			context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
				Limit:     aws.Int32(50),
				NextToken: nextToken,
			},
		)

		if err != nil {
			inst.logger.Println(err)
			break
		}

		nextToken = output.NextToken
		inst.allLogGroups = append(inst.allLogGroups, output.LogGroups...)
		if nextToken == nil {
			break
		}
	}

	return inst.allLogGroups
}

func (inst *CloudWatchLogsApi) FilterGroupByName(name string) []types.LogGroup {

	if len(inst.allLogGroups) < 1 {
		inst.ListLogGroups(true)
	}

	var found_groups []types.LogGroup

	for _, info := range inst.allLogGroups {
		found := strings.Contains(*info.LogGroupName, name)
		if found {
			found_groups = append(found_groups, info)
		}
	}
	return found_groups
}

func (inst *CloudWatchLogsApi) ListLogStreams(
	logGroupName string,
	searchPrefix string,
	reset bool,
) []types.LogStream {
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

	var empty = make([]types.LogStream, 0)
	if !inst.logStreamsPaginator.HasMorePages() {
		return empty
	}

	var output, err = inst.logStreamsPaginator.NextPage(context.TODO())

	if err != nil {
		inst.logger.Println(err)
		return empty
	}

	return output.LogStreams
}

func (inst *CloudWatchLogsApi) ListLogEvents(
	logGroupName string,
	logStreamName string,
	reset bool,
) []types.OutputLogEvent {

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

	var empty = make([]types.OutputLogEvent, 0)
	if !inst.logEventsPaginator.HasMorePages() {
		return empty
	}

	var output, err = inst.logEventsPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return empty
	}

	return output.Events
}

func (inst *CloudWatchLogsApi) ListFilteredLogEvents(
	logGroupName string,
	startDateTime time.Time,
	endDateTime time.Time,
	nextToken *string,
) ([]types.FilteredLogEvent, *string) {
	output, err := inst.client.FilterLogEvents(
		context.TODO(),
		&cloudwatchlogs.FilterLogEventsInput{
			Limit:        aws.Int32(500),
			LogGroupName: aws.String(logGroupName),
			StartTime:    aws.Int64(startDateTime.UnixMilli()),
			EndTime:      aws.Int64(endDateTime.UnixMilli()),
			NextToken:    nextToken,
		},
	)

	if err != nil {
		inst.logger.Println(err)
		var empty = make([]types.FilteredLogEvent, 0)
		return empty, nil
	}

	return output.Events, output.NextToken
}

func (inst *CloudWatchLogsApi) StartInightsQuery(
	logGroups []string,
	startTime time.Time,
	endTime time.Time,
	query string,
) string {
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
		return ""
	}

	return aws.ToString(output.QueryId)
}

func (inst *CloudWatchLogsApi) GetInightsQueryResults(
	queryId string,
) ([][]types.ResultField, types.QueryStatus) {
	var output, err = inst.client.GetQueryResults(
		context.TODO(), &cloudwatchlogs.GetQueryResultsInput{
			QueryId: aws.String(queryId),
		})

	var empty [][]types.ResultField
	if err != nil {
		inst.logger.Println(err)
		return empty, types.QueryStatusUnknown
	}

	return output.Results, output.Status
}

func (inst *CloudWatchLogsApi) GetInsightsLogRecord(
	recordPtr string,
) map[string]string {
	var output, err = inst.client.GetLogRecord(
		context.TODO(), &cloudwatchlogs.GetLogRecordInput{
			LogRecordPointer: aws.String(recordPtr),
		})

	var empty = make(map[string]string, 0)
	if err != nil {
		inst.logger.Println(err)
		return empty
	}

	return output.LogRecord
}
