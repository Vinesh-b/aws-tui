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
		{"ARN", "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"},
		{"KMS Key", "kms-1"},
		{"Class", "STANDARD"},
		{"Retention (days)", "30"},
		{"Created", "2024-01-02 03:04:05"},
	})
}

func TestLogGroupDetailsTable__MissingValuesShowZeroDefaults(t *testing.T) {
	var table = NewLogGroupDetailsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", ""},
		{"ARN", ""},
		{"KMS Key", ""},
		{"Class", ""},
		{"Retention (days)", "Never expire"},
		{"Created", ""},
	})
}
