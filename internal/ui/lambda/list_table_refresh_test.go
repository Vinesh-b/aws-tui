package lambda

import (
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/lambda/lambdafake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func newRunningListTable(t *testing.T, api awsapi.LambdaApi) *LambdaListTable {
	var serviceCtx = uitest.NewServiceContextWith(api)
	var table = NewLambdasListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func TestListTable__RefreshLoadsFunctionsFromApi(t *testing.T) {
	var api = lambdafake.New(lambdafake.WithFunctions([]types.FunctionConfiguration{newFunction("alpha"), newFunction("beta")}))
	var table = newRunningListTable(t, api)

	table.RefreshLambdas(true)

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	uitest.OnUI(t, table.serviceCtx.App, func() {
		uitest.AssertTableRows(t, table, []core.TableRow{
			{"Name", "LastModified"},
			{"alpha", "2024-01-02T03:04:05.000+0000"},
			{"beta", "2024-01-02T03:04:05.000+0000"},
		})
	})
}

func TestListTable__RefreshReportsApiError(t *testing.T) {
	var api = lambdafake.New(lambdafake.WithErr(errors.New("access denied")))
	var table = newRunningListTable(t, api)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshLambdas(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestListTable__SearchFiltersToTheMatchingFunctions(t *testing.T) {
	var api = lambdafake.New(lambdafake.WithFunctions([]types.FunctionConfiguration{newFunction("orders"), newFunction("billing")}))
	var table = newRunningListTable(t, api)
	table.RefreshLambdas(true)
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, table.serviceCtx.App, func() { table.SetSearchText("bill") })

	uitest.Eventually(t, table.serviceCtx.App, func() bool {
		return table.GetTable().GetRowCount() == 2 && table.GetCellText(1, 0) == "billing"
	})
}
