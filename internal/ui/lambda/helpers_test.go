package lambda

import (
	"context"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/rivo/tview"
)

func newTestServiceContext() *core.ServiceContext[awsapi.LambdaApi] {
	var appCtx = core.NewAppContext(
		context.Background(), tview.NewApplication(), nil, nil, &core.AppTheme{},
	)
	return core.NewServiceViewContext[awsapi.LambdaApi](appCtx, nil)
}

func assertRows(t *testing.T, table *core.DetailsTable, want []core.TableRow) {
	t.Helper()
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if extra := table.GetCellText(len(want), 0); extra != "" {
		t.Errorf("unexpected extra row starting with %q", extra)
	}
}

func newFunction(name string) types.FunctionConfiguration {
	return types.FunctionConfiguration{
		FunctionName: aws.String(name),
		LastModified: aws.String("2024-01-02T03:04:05.000+0000"),
	}
}
