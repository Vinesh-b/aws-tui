package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestLogGroupDetailsTable__PopulatesFromLogGroup(t *testing.T) {
	var table = NewLogGroupDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	table.data = types.LogGroup{
		LogGroupName:    aws.String("/aws/lambda/orders"),
		LogGroupArn:     aws.String("arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"),
		KmsKeyId:        aws.String("kms-1"),
		LogGroupClass:   types.LogGroupClassStandard,
		RetentionInDays: aws.Int32(30),
		CreationTime:    millisAt(0),
	}

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "/aws/lambda/orders"},
		{"Arn", "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"},
		{"KMSKey", "kms-1"},
		{"Class", "STANDARD"},
		{"RetentionDays", "30"},
		{"CreatedTime", "2024-01-02 03:04:05"},
	})
}

func TestLogGroupDetailsTable__MissingValuesShowZeroDefaults(t *testing.T) {
	var table = NewLogGroupDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"Arn", ""},
		{"KMSKey", ""},
		{"Class", ""},
		{"RetentionDays", "0"},
		{"CreatedTime", "1970-01-01 00:00:00"},
	})
}
