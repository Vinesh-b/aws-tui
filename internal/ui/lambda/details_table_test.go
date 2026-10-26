package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestDetailsTable__EmptyConfigShowsBlankValues(t *testing.T) {
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Description", ""},
		{"ARN", ""},
		{"Version", ""},
		{"State", ""},
		{"Last Modified", ""},
		{"Role", ""},
		{"Configuration"},
		{"Runtime", ""},
		{"Architectures", ""},
		{"Memory Size (MB)", ""},
		{"Timeout (s)", ""},
	})
}

func TestDetailsTable__PopulatesFromConfig(t *testing.T) {
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{
		FunctionName:  aws.String("orders"),
		Description:   aws.String("Handles orders"),
		FunctionArn:   aws.String("arn:aws:lambda:us-east-1:123456789012:function:orders"),
		Version:       aws.String("$LATEST"),
		MemorySize:    aws.Int32(256),
		Runtime:       types.RuntimeGo1x,
		Architectures: []types.Architecture{types.ArchitectureArm64},
		Timeout:       aws.Int32(30),
		State:         types.StateActive,
		LastModified:  aws.String("2024-01-02T03:04:05.000+0000"),
		Role:          aws.String("arn:aws:iam::123456789012:role/orders"),
		LoggingConfig: &types.LoggingConfig{
			LogGroup:       aws.String("/aws/lambda/orders"),
			SystemLogLevel: types.SystemLogLevelWarn,
		},
	}

	table.populateLambdaDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Description", "Handles orders"},
		{"ARN", "arn:aws:lambda:us-east-1:123456789012:function:orders"},
		{"Version", "$LATEST"},
		{"State", "Active"},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"Role", "arn:aws:iam::123456789012:role/orders"},
		{"Configuration"},
		{"Runtime", "go1.x"},
		{"Architectures", "arm64"},
		{"Memory Size (MB)", "256"},
		{"Timeout (s)", "30"},
		{"Logging"},
		{"Log Group", "/aws/lambda/orders"},
		{"System Log Level", "WARN"},
	})
}
