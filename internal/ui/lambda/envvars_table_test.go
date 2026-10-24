package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestEnvVarsTable__NoEnvironmentIsEmpty(t *testing.T) {
	var table = NewLambdaEnvVarsTable(uitest.NewServiceContext[awsapi.LambdaApi]())

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestEnvVarsTable__SortsVariablesByName(t *testing.T) {
	var table = NewLambdaEnvVarsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{
		Environment: &types.EnvironmentResponse{
			Variables: map[string]string{
				"STAGE":     "prod",
				"BUCKET":    "orders-bucket",
				"LOG_LEVEL": "info",
			},
		},
	}

	table.populateLambdaEnvVarsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"BUCKET", "orders-bucket"},
		{"LOG_LEVEL", "info"},
		{"STAGE", "prod"},
	})
}
