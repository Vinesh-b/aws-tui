package awsapi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudWatchLogsApi interface {
	ListLogGroups(ctx context.Context, reset bool) ([]types.LogGroup, error)
	ListLogStreams(
		ctx context.Context,
		logGroupName string,
		searchPrefix string,
		reset bool,
	) ([]types.LogStream, error)
	ListLogEvents(
		ctx context.Context,
		logGroupName string,
		logStreamName string,
		reset bool,
	) ([]types.OutputLogEvent, error)
	ListFilteredLogEvents(
		ctx context.Context,
		logGroupName string,
		startDateTime time.Time,
		endDateTime time.Time,
		reset bool,
	) ([]types.FilteredLogEvent, error)
	StartInsightsQuery(
		ctx context.Context,
		logGroups []string,
		startTime time.Time,
		endTime time.Time,
		query string,
	) (string, error)
	StopInsightsQuery(
		ctx context.Context,
		queryId string,
	) (bool, error)
	GetInsightsQueryResults(
		ctx context.Context,
		queryId string,
	) ([][]types.ResultField, types.QueryStatus, error)
	GetInsightsLogRecord(
		ctx context.Context,
		recordPtr string,
	) (map[string]string, error)
}

type cloudWatchLogsApi struct {
	baseApi
	logEventsPaginator         *cloudwatchlogs.GetLogEventsPaginator
	logStreamsPaginator        *cloudwatchlogs.DescribeLogStreamsPaginator
	logGroupsPaginator         *cloudwatchlogs.DescribeLogGroupsPaginator
	filteredLogEventsPaginator *cloudwatchlogs.FilterLogEventsPaginator
}

func NewCloudWatchLogsApi(
	logger *log.Logger,
) CloudWatchLogsApi {
	return &cloudWatchLogsApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *cloudWatchLogsApi) ListLogGroups(ctx context.Context, reset bool) ([]types.LogGroup, error) {
	if reset || inst.logGroupsPaginator == nil {
		inst.logGroupsPaginator = cloudwatchlogs.NewDescribeLogGroupsPaginator(
			GetAwsApiClients().cloudwatchlogs,
			&cloudwatchlogs.DescribeLogGroupsInput{
				Limit: aws.Int32(50),
			},
		)
	}

	var result, err = collectPaginator(inst.logGroupsPaginator.HasMorePages, func() ([]types.LogGroup, error) {
		var output, err = inst.logGroupsPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		return output.LogGroups, nil
	})

	sortByKey(result, func(g types.LogGroup) *string { return g.LogGroupName })

	return result, inst.logErr(err)
}

func (inst *cloudWatchLogsApi) ListLogStreams(
	ctx context.Context,
	logGroupName string,
	searchPrefix string,
	reset bool,
) ([]types.LogStream, error) {
	var empty = []types.LogStream{}

	if err := requireArg("log group", logGroupName); err != nil {
		return empty, err
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

		var client = GetAwsApiClients().cloudwatchlogs

		inst.logStreamsPaginator = cloudwatchlogs.NewDescribeLogStreamsPaginator(
			client,
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

	var output, err = inst.logStreamsPaginator.NextPage(ctx)

	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.LogStreams, nil
}

func (inst *cloudWatchLogsApi) ListLogEvents(
	ctx context.Context,
	logGroupName string,
	logStreamName string,
	reset bool,
) ([]types.OutputLogEvent, error) {
	var empty = []types.OutputLogEvent{}

	if err := requireArg("log group", logGroupName); err != nil {
		return empty, err
	}

	if err := requireArg("log stream", logStreamName); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	if reset || inst.logEventsPaginator == nil {
		inst.logEventsPaginator = cloudwatchlogs.NewGetLogEventsPaginator(
			client,
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

	var output, err = inst.logEventsPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.Events, nil
}

func (inst *cloudWatchLogsApi) ListFilteredLogEvents(
	ctx context.Context,
	logGroupName string,
	startDateTime time.Time,
	endDateTime time.Time,
	reset bool,
) ([]types.FilteredLogEvent, error) {
	var empty = []types.FilteredLogEvent{}

	if err := requireArg("log group", logGroupName); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	if reset || inst.filteredLogEventsPaginator == nil {
		inst.filteredLogEventsPaginator = cloudwatchlogs.NewFilterLogEventsPaginator(
			client,
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

	var output, err = inst.filteredLogEventsPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.Events, nil
}

func (inst *cloudWatchLogsApi) StartInsightsQuery(
	ctx context.Context,
	logGroups []string,
	startTime time.Time,
	endTime time.Time,
	query string,
) (string, error) {
	var client = GetAwsApiClients().cloudwatchlogs
	var output, err = client.StartQuery(
		ctx, &cloudwatchlogs.StartQueryInput{
			StartTime:     aws.Int64(startTime.Unix()),
			EndTime:       aws.Int64(endTime.Unix()),
			LogGroupNames: logGroups,
			QueryString:   aws.String(query),
		},
	)

	if err != nil {
		return "", inst.logErr(err)
	}

	return aws.ToString(output.QueryId), nil
}

func (inst *cloudWatchLogsApi) StopInsightsQuery(
	ctx context.Context,
	queryId string,
) (bool, error) {
	if err := requireArg("query ID", queryId); err != nil {
		return false, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	var output, err = client.StopQuery(
		ctx, &cloudwatchlogs.StopQueryInput{
			QueryId: aws.String(queryId),
		},
	)

	if err != nil {
		return false, inst.logErr(err)
	}

	return *aws.Bool(output.Success), nil
}

func (inst *cloudWatchLogsApi) GetInsightsQueryResults(
	ctx context.Context,
	queryId string,
) ([][]types.ResultField, types.QueryStatus, error) {
	var empty [][]types.ResultField

	if err := requireArg("query ID", queryId); err != nil {
		return empty, types.QueryStatusUnknown, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	var output, err = client.GetQueryResults(
		ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: aws.String(queryId),
		})

	if err != nil {
		return empty, types.QueryStatusUnknown, inst.logErr(err)
	}

	return output.Results, output.Status, nil
}

func (inst *cloudWatchLogsApi) GetInsightsLogRecord(
	ctx context.Context,
	recordPtr string,
) (map[string]string, error) {
	var empty = map[string]string{}

	if err := requireArg("record pointer", recordPtr); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	var output, err = client.GetLogRecord(
		ctx, &cloudwatchlogs.GetLogRecordInput{
			LogRecordPointer: aws.String(recordPtr),
		})

	if err != nil {
		return empty, inst.logErr(err)
	}

	return output.LogRecord, nil
}
