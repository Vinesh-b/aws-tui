package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestLogStreamDetailsTable__PopulatesFromLogStream(t *testing.T) {
	var table = NewLogStreamDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = types.LogStream{
		LogStreamName:       aws.String("2024/01/02/[$LATEST]abc"),
		Arn:                 aws.String("arn:aws:logs:us-east-1:123456789012:log-group:g:log-stream:s"),
		FirstEventTimestamp: millisAt(1),
		LastEventTimestamp:  millisAt(2),
		LastIngestionTime:   millisAt(3),
		CreationTime:        millisAt(0),
	}

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "2024/01/02/[$LATEST]abc"},
		{"ARN", "arn:aws:logs:us-east-1:123456789012:log-group:g:log-stream:s"},
		{"First Event", "2024-01-02 03:04:06"},
		{"Last Event", "2024-01-02 03:04:07"},
		{"Last Ingestion", "2024-01-02 03:04:08"},
		{"Created", "2024-01-02 03:04:05"},
	})
}
