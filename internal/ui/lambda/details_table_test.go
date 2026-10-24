package lambda

import (
	"testing"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestDetailsTable__EmptyConfigShowsBlankValues(t *testing.T) {
	var table = NewLambdaDetailsTable(newTestServiceContext())

	assertRows(t, table.DetailsTable, []core.TableRow{
		{"Description", ""},
		{"Arn", ""},
		{"Version", ""},
		{"MemorySize", "0"},
		{"Runtime", ""},
		{"Arch", "[]"},
		{"Timeout", "0"},
		{"LoggingGroup", ""},
		{"SystemLogLevel", ""},
		{"State", ""},
		{"LastModified", ""},
		{"Role", ""},
	})
}

func TestDetailsTable__PopulatesFromConfig(t *testing.T) {
	var table = NewLambdaDetailsTable(newTestServiceContext())
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

	assertRows(t, table.DetailsTable, []core.TableRow{
		{"Description", "Handles orders"},
		{"Arn", "arn:aws:lambda:us-east-1:123456789012:function:orders"},
		{"Version", "$LATEST"},
		{"MemorySize", "256"},
		{"Runtime", "go1.x"},
		{"Arch", "[arm64]"},
		{"Timeout", "30"},
		{"LoggingGroup", "/aws/lambda/orders"},
		{"SystemLogLevel", "WARN"},
		{"State", "Active"},
		{"LastModified", "2024-01-02T03:04:05.000+0000"},
		{"Role", "arn:aws:iam::123456789012:role/orders"},
	})
}
