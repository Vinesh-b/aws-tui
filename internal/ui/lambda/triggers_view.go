package lambda

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type eventSourceMapping = types.EventSourceMappingConfiguration

// eventSource is the type and name of what a mapping reads from, such as
// "SQS" and the name of the queue.
func eventSource(mapping eventSourceMapping) (string, string) {
	var source, err = arn.Parse(aws.ToString(mapping.EventSourceArn))
	if err != nil {
		if mapping.SelfManagedEventSource != nil {
			return "Kafka", strings.Join(mapping.Topics, ", ")
		}
		return "", aws.ToString(mapping.EventSourceArn)
	}

	var parts = strings.FieldsFunc(source.Resource, func(r rune) bool { return r == '/' || r == ':' })
	var partAfter = func(prefix string) string {
		if idx := slices.Index(parts, prefix); idx >= 0 && idx+1 < len(parts) {
			return parts[idx+1]
		}
		return source.Resource
	}

	switch source.Service {
	case "sqs":
		return "SQS", source.Resource
	case "dynamodb":
		return "DynamoDB Stream", partAfter("table")
	case "kinesis":
		return "Kinesis", partAfter("stream")
	case "kafka":
		return "MSK", partAfter("cluster")
	case "mq":
		return "Amazon MQ", partAfter("broker")
	case "rds":
		return "DocumentDB", partAfter("cluster")
	}
	return source.Service, source.Resource
}

// linkableArn is the ARN if it is of a resource another service can open
func linkableArn(text string) string {
	if parsed, err := arn.Parse(text); err == nil && parsed.Service == "sqs" {
		return text
	}
	return ""
}

func eventSourceRows(mapping eventSourceMapping) []core.TableRow {
	var sourceType, sourceName = eventSource(mapping)
	var sourceArn = aws.ToString(mapping.EventSourceArn)

	var rows = []core.TableRow{
		core.DetailsLink("Source", sourceName, linkableArn(sourceArn)),
		{"Type", sourceType},
		{"Source ARN", sourceArn},
		{"State", aws.ToString(mapping.State)},
		{"State Reason", aws.ToString(mapping.StateTransitionReason)},
		{"Last Result", aws.ToString(mapping.LastProcessingResult)},
		{"Last Modified", core.FormatTime(mapping.LastModified)},
		{"UUID", aws.ToString(mapping.UUID)},
	}

	if len(mapping.Topics) > 0 {
		rows = append(rows, core.TableRow{"Topics", strings.Join(mapping.Topics, ", ")})
	}

	rows = append(rows, core.DetailsSection("Batching",
		core.TableRow{"Batch Size", core.FormatOptionalInt(mapping.BatchSize)},
		core.TableRow{"Max Batching Window (s)", core.FormatOptionalInt(mapping.MaximumBatchingWindowInSeconds)},
		core.TableRow{"Starting Position", string(mapping.StartingPosition)},
		core.TableRow{"Parallelization Factor", core.FormatOptionalInt(mapping.ParallelizationFactor)},
	)...)

	var onFailure = ""
	if mapping.DestinationConfig != nil && mapping.DestinationConfig.OnFailure != nil {
		onFailure = aws.ToString(mapping.DestinationConfig.OnFailure.Destination)
	}
	var bisect = ""
	if mapping.BisectBatchOnFunctionError != nil {
		bisect = strconv.FormatBool(*mapping.BisectBatchOnFunctionError)
	}
	var reportFailures = slices.Contains(mapping.FunctionResponseTypes, types.FunctionResponseTypeReportBatchItemFailures)

	rows = append(rows, core.DetailsSection("Errors",
		core.TableRow{"Max Retry Attempts", core.FormatOptionalInt(mapping.MaximumRetryAttempts)},
		core.TableRow{"Max Record Age (s)", core.FormatOptionalInt(mapping.MaximumRecordAgeInSeconds)},
		core.TableRow{"Split Batch On Error", bisect},
		core.TableRow{"Report Batch Item Failures", strconv.FormatBool(reportFailures)},
		core.DetailsLink("On Failure", onFailure, linkableArn(onFailure)),
	)...)

	if mapping.FilterCriteria != nil {
		var filters []core.TableRow
		for _, filter := range mapping.FilterCriteria.Filters {
			filters = append(filters, core.TableRow{"Pattern", aws.ToString(filter.Pattern)})
		}
		rows = append(rows, core.DetailsSection("Filters", filters...)...)
	}

	return rows
}

// NewLambdaTriggersView lists the event sources that invoke the lambda beside
// the details of the selected one.
func NewLambdaTriggersView(serviceCtx *core.ServiceContext[awsapi.LambdaApi]) *LambdaItemsView[eventSourceMapping] {
	return NewLambdaItemsView(serviceCtx, "Triggers", "Trigger Details",
		core.TableRow{"Source", "Type", "State", "Batch Size"},
		func(mapping eventSourceMapping) core.TableRow {
			var sourceType, sourceName = eventSource(mapping)
			return core.TableRow{
				sourceName,
				sourceType,
				aws.ToString(mapping.State),
				core.FormatOptionalInt(mapping.BatchSize),
			}
		},
		func(mapping eventSourceMapping) string {
			var _, sourceName = eventSource(mapping)
			return sourceName
		},
		eventSourceRows,
		func(ctx context.Context, lambdaArn string) ([]eventSourceMapping, error) {
			return serviceCtx.Api.ListEventSourceMappings(ctx, lambdaArn)
		},
	)
}

// lambdaName is the name in a lambda ARN, or the text if it isn't one
func lambdaName(lambdaArn string) string {
	if parsed, err := arn.Parse(lambdaArn); err == nil {
		if name, ok := core.ArnResourceName(parsed, "function"); ok {
			return name
		}
	}
	return lambdaArn
}
