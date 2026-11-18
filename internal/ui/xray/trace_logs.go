package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

// traceLogsMargin is how long before and after a trace its logs are looked
// for, as they are written a little before and after the segments are timed
const traceLogsMargin = 5 * time.Minute

// lambdaFunctionType is the type and origin of the segments of a Lambda
// function, whose logs are in its default log group
const lambdaFunctionType = "AWS::Lambda::Function"

// segmentLogsDocument is the part of the json document of a segment that names
// the log groups it logs to, which the CloudWatch agent of ECS, EC2 and
// Elastic Beanstalk adds
type segmentLogsDocument struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Aws    struct {
		CloudwatchLogs []struct {
			LogGroup string `json:"log_group"`
		} `json:"cloudwatch_logs"`
	} `json:"aws"`
}

func lambdaLogGroup(function string) string {
	return "/aws/lambda/" + function
}

// traceLogGroups are the log groups the services of the trace log to, sorted:
// the default log group of its Lambda functions, and those its segments name.
// The segments are nil until they are loaded.
func traceLogGroups(summary types.TraceSummary, trace *types.Trace) []string {
	var groups []string
	for _, resource := range summary.ResourceARNs {
		var parsed, err = arn.Parse(aws.ToString(resource.ARN))
		if err != nil || parsed.Service != "lambda" {
			continue
		}
		if function, ok := core.ArnResourceName(parsed, "function"); ok {
			groups = append(groups, lambdaLogGroup(function))
		}
	}
	for _, service := range summary.ServiceIds {
		if aws.ToString(service.Type) == lambdaFunctionType && aws.ToString(service.Name) != "" {
			groups = append(groups, lambdaLogGroup(aws.ToString(service.Name)))
		}
	}

	if trace != nil {
		for _, segment := range trace.Segments {
			var document segmentLogsDocument
			if err := json.Unmarshal([]byte(aws.ToString(segment.Document)), &document); err != nil {
				continue
			}
			if document.Origin == lambdaFunctionType && document.Name != "" {
				groups = append(groups, lambdaLogGroup(document.Name))
			}
			for _, logs := range document.Aws.CloudwatchLogs {
				if logs.LogGroup != "" {
					groups = append(groups, logs.LogGroup)
				}
			}
		}
	}

	slices.Sort(groups)
	return slices.Compact(groups)
}

// traceLogsQuery is the Logs Insights query of the log lines that name the
// trace, in the log groups of its services, from a little before it started
// to a little after it ended
func traceLogsQuery(summary types.TraceSummary, trace *types.Trace) (core.LogsQuery, error) {
	var traceId = aws.ToString(summary.Id)
	if traceId == "" {
		return core.LogsQuery{}, errors.New("No trace is selected")
	}
	if summary.StartTime == nil {
		return core.LogsQuery{}, fmt.Errorf("Trace %s has no start time to find its logs by", traceId)
	}
	var logGroups = traceLogGroups(summary, trace)
	if len(logGroups) == 0 {
		return core.LogsQuery{}, fmt.Errorf("No log groups were found for trace %s", traceId)
	}

	var start = *summary.StartTime
	var duration = time.Duration(aws.ToFloat64(summary.Duration) * float64(time.Second))
	return core.LogsQuery{
		Query: "fields @timestamp, @log, @message\n" +
			fmt.Sprintf("| filter @message like %q\n", traceId) +
			"| sort @timestamp asc\n" +
			"| limit 1000",
		LogGroups: logGroups,
		Start:     start.Add(-traceLogsMargin),
		End:       start.Add(duration + traceLogsMargin),
	}, nil
}
