package cloudwatchlogs

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

	var nextToken *string = nil

	for {
		output, err := inst.client.DescribeLogGroups(
			context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
				Limit:     aws.Int32(50),
				NextToken: nextToken,
			},
		)

		if err != nil {
			log.Println(err)
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
	searchPrefix *string,
	reset bool,
) []types.LogStream {

	if reset || inst.logStreamsPaginator == nil {
		if searchPrefix != nil && len(*searchPrefix) == 0 {
			searchPrefix = nil
		}
		inst.logStreamsPaginator = cloudwatchlogs.NewDescribeLogStreamsPaginator(
			inst.client,
			&cloudwatchlogs.DescribeLogStreamsInput{
				Descending:          aws.Bool(true),
				Limit:               aws.Int32(50),
				LogGroupName:        aws.String(logGroupName),
				LogStreamNamePrefix: searchPrefix,
			},
		)
	}

	var empty = make([]types.LogStream, 0)
	if !inst.logStreamsPaginator.HasMorePages() {
		return empty
	}

	var output, err = inst.logStreamsPaginator.NextPage(context.TODO())

	if err != nil {
		log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
		var empty = make([]types.FilteredLogEvent, 0)
		return empty, nil
	}

	return output.Events, output.NextToken
}
