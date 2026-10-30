package ssm

import (
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/ssm/ssmfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestParametersTable__RefreshLoadsParametersFromApi(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithParameters([]types.Parameter{newParameter("/app/db/host", 3)}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var table = NewSSMParametersListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshParameters("/", true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		parameterHeadings,
		{"/app/db/host", "String", "3", "2024-01-02 03:04:05"},
	})
}

func TestParametersTable__RefreshReportsApiError(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var table = NewSSMParametersListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshParameters("/", true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestParametersTable__SearchShowsMatchingParameters(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithParameters([]types.Parameter{
		newParameter("/app/db/host", 1), newParameter("/app/cache/url", 1),
	}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var table = NewSSMParametersListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshParameters("/", true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("cache") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		parameterHeadings,
		{"/app/cache/url", "String", "1", "2024-01-02 03:04:05"},
	})
}

func TestParameterHistoryTable__RefreshLoadsHistoryFromApi(t *testing.T) {
	var api = ssmfake.New(ssmfake.WithHistory([]types.ParameterHistory{newHistory(2, "new"), newHistory(1, "old")}))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.SystemsManagerApi](api)
	var table = NewSSMParameterHistoryTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshHistory(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		historyHeadings,
		{"2", "SecureString", "new", "2024-01-02 03:04:05", "arn:aws:iam::123456789012:user/alice"},
		{"1", "SecureString", "old", "2024-01-02 03:04:05", "arn:aws:iam::123456789012:user/alice"},
	})
}
