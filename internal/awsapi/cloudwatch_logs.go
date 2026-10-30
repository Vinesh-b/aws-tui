package awsapi

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// InsightsQueryResults are the results of a Logs Insights query, with how far
// it has got
type InsightsQueryResults struct {
	Results    [][]types.ResultField
	Status     types.QueryStatus
	Statistics *types.QueryStatistics
}

type CloudWatchLogsApi interface {
	ListLogGroups(ctx context.Context, reset bool) ([]types.LogGroup, error)
	ListLogStreams(
		ctx context.Context,
		logGroupName string,
		searchPrefix string,
		reset bool,
	) ([]types.LogStream, error)
	// ListLogEvents pages the events of a stream from the oldest, or from
	// startTime if that is not zero.
	ListLogEvents(
		ctx context.Context,
		logGroupName string,
		logStreamName string,
		startTime time.Time,
		reset bool,
	) ([]types.OutputLogEvent, error)
	// FollowLogEvents gives the events that came after those ListLogEvents
	// gave last, for a stream that is being followed. It is empty until
	// ListLogEvents has given its first page.
	FollowLogEvents(
		ctx context.Context,
		logGroupName string,
		logStreamName string,
	) ([]types.OutputLogEvent, error)
	// ListFilteredLogEvents pages the events of a stream, or of the whole
	// log group if the stream is empty, that match the filter pattern. They
	// are from startTime if that is not zero.
	ListFilteredLogEvents(
		ctx context.Context,
		logGroupName string,
		logStreamName string,
		filterPattern string,
		startTime time.Time,
		reset bool,
	) ([]types.OutputLogEvent, error)
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
	// GetInsightsQueryResults gives the status of the query and the results it
	// has so far, which are all of them once it is complete.
	GetInsightsQueryResults(
		ctx context.Context,
		queryId string,
	) (InsightsQueryResults, error)
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
	// logEventsMtx keeps the paging of the events and the following of them
	// from running together, as both move logEventsToken on
	logEventsMtx   sync.Mutex
	logEventsToken *string
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
	startTime time.Time,
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

	inst.logEventsMtx.Lock()
	defer inst.logEventsMtx.Unlock()

	if reset || inst.logEventsPaginator == nil {
		var startTimePtr *int64 = nil
		if !startTime.IsZero() {
			startTimePtr = aws.Int64(startTime.UnixMilli())
		}

		inst.logEventsToken = nil
		inst.logEventsPaginator = cloudwatchlogs.NewGetLogEventsPaginator(
			client,
			&cloudwatchlogs.GetLogEventsInput{
				LogStreamName: aws.String(logStreamName),
				Limit:         aws.Int32(500),
				LogGroupName:  aws.String(logGroupName),
				StartFromHead: aws.Bool(true),
				StartTime:     startTimePtr,
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

	inst.logEventsToken = output.NextForwardToken
	return output.Events, nil
}

func (inst *cloudWatchLogsApi) FollowLogEvents(
	ctx context.Context,
	logGroupName string,
	logStreamName string,
) ([]types.OutputLogEvent, error) {
	var empty = []types.OutputLogEvent{}

	if err := requireArg("log group", logGroupName); err != nil {
		return empty, err
	}

	if err := requireArg("log stream", logStreamName); err != nil {
		return empty, err
	}

	inst.logEventsMtx.Lock()
	defer inst.logEventsMtx.Unlock()

	if inst.logEventsToken == nil {
		return empty, nil
	}

	// The paginator stops once the end of the stream is reached, but the
	// token of the end can be asked again for what came after it
	var output, err = GetAwsApiClients().cloudwatchlogs.GetLogEvents(
		ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(logGroupName),
			LogStreamName: aws.String(logStreamName),
			Limit:         aws.Int32(500),
			StartFromHead: aws.Bool(true),
			NextToken:     inst.logEventsToken,
		})
	if err != nil {
		return empty, inst.logErr(err)
	}

	if output.NextForwardToken != nil {
		inst.logEventsToken = output.NextForwardToken
	}
	return output.Events, nil
}

func (inst *cloudWatchLogsApi) ListFilteredLogEvents(
	ctx context.Context,
	logGroupName string,
	logStreamName string,
	filterPattern string,
	startTime time.Time,
	reset bool,
) ([]types.OutputLogEvent, error) {
	var empty = []types.OutputLogEvent{}

	if err := requireArg("log group", logGroupName); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	inst.logEventsMtx.Lock()
	defer inst.logEventsMtx.Unlock()

	if reset || inst.filteredLogEventsPaginator == nil {
		var input = &cloudwatchlogs.FilterLogEventsInput{
			Limit:        aws.Int32(500),
			LogGroupName: aws.String(logGroupName),
		}
		if len(logStreamName) > 0 {
			input.LogStreamNames = []string{logStreamName}
		}
		if len(filterPattern) > 0 {
			input.FilterPattern = aws.String(filterPattern)
		}
		if !startTime.IsZero() {
			input.StartTime = aws.Int64(startTime.UnixMilli())
		}
		inst.filteredLogEventsPaginator = cloudwatchlogs.NewFilterLogEventsPaginator(client, input)
	}
	if !inst.filteredLogEventsPaginator.HasMorePages() {
		return empty, nil
	}

	var output, err = inst.filteredLogEventsPaginator.NextPage(ctx)
	if err != nil {
		return empty, inst.logErr(err)
	}

	var events = make([]types.OutputLogEvent, 0, len(output.Events))
	for _, event := range output.Events {
		events = append(events, types.OutputLogEvent{
			Message:       event.Message,
			Timestamp:     event.Timestamp,
			IngestionTime: event.IngestionTime,
		})
	}
	return events, nil
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
) (InsightsQueryResults, error) {
	var empty = InsightsQueryResults{Status: types.QueryStatusUnknown}

	if err := requireArg("query ID", queryId); err != nil {
		return empty, err
	}

	var client = GetAwsApiClients().cloudwatchlogs

	var output, err = client.GetQueryResults(
		ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: aws.String(queryId),
		})

	if err != nil {
		return empty, inst.logErr(err)
	}

	return InsightsQueryResults{
		Results:    output.Results,
		Status:     output.Status,
		Statistics: output.Statistics,
	}, nil
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
