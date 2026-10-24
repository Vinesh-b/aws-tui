package lambda

import (
	"testing"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestEnvVarsTable__NoEnvironmentIsEmpty(t *testing.T) {
	var table = NewLambdaEnvVarsTable(newTestServiceContext())

	assertRows(t, table.DetailsTable, nil)
}

func TestEnvVarsTable__SortsVariablesByName(t *testing.T) {
	var table = NewLambdaEnvVarsTable(newTestServiceContext())
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

	assertRows(t, table.DetailsTable, []core.TableRow{
		{"BUCKET", "orders-bucket"},
		{"LOG_LEVEL", "info"},
		{"STAGE", "prod"},
	})
}
