package lambda

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestListTable__OpenLambdaLoadsListToSelectIt(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("alpha"), newFunction("beta")}}
	var table = newRunningListTable(t, api)
	var opened = false

	uitest.OnUI(t, table.serviceCtx.App, func() { table.OpenLambda("beta", func() { opened = true }) })

	uitest.Eventually(t, table.serviceCtx.App, func() bool { return opened })
	uitest.OnUI(t, table.serviceCtx.App, func() {
		if got := aws.ToString(table.GetSelectedLambda().FunctionName); got != "beta" {
			t.Errorf("selected = %q, want beta", got)
		}
	})
}

func TestListTable__OpenLoadedLambdaDoesNotLoadAgain(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("alpha"), newFunction("beta")}}
	var table = newRunningListTable(t, api)
	table.RefreshLambdas(true)
	uitest.Eventually(t, table.serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	var opened = false

	uitest.OnUI(t, table.serviceCtx.App, func() { table.OpenLambda("alpha", func() { opened = true }) })

	if !opened {
		t.Error("loaded lambda should be opened straight away")
	}
	if got := api.listCalls.Load(); got != 1 {
		t.Errorf("list calls = %d, want 1", got)
	}
}

func TestListTable__OpenMissingLambdaReportsIt(t *testing.T) {
	var api = &fakeLambdaApi{functions: []types.FunctionConfiguration{newFunction("alpha")}}
	var table = newRunningListTable(t, api)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, table.serviceCtx.App, func() { table.OpenLambda("gamma", func() { t.Error("opened a missing lambda") }) })

	if got := uitest.AwaitError(t, received); got != "Lambda gamma not found" {
		t.Errorf("message = %q", got)
	}
}

func TestHomeView__OpensLambdaResources(t *testing.T) {
	var page = NewLambdaHomeView(uitest.NewAppContext(), &fakeLambdaApi{}, awsapi.CloudWatchLogsApi(nil), awsapi.CloudWatchMetricsApi(nil))

	var opener, ok = page.(core.ResourceOpener)

	var function, _ = arn.Parse("arn:aws:lambda:us-east-1:123456789012:function:orders")
	if !ok || !opener.OpensArn(function) {
		t.Error("lambda page should open lambda resources")
	}
}

func TestDetailsTable__LogGroupLinksToLogGroup(t *testing.T) {
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{
		FunctionArn:   aws.String("arn:aws:lambda:us-east-1:123456789012:function:orders"),
		LoggingConfig: &types.LoggingConfig{LogGroup: aws.String("/aws/lambda/orders")},
	}

	table.populateLambdaDetailsTable()

	const logGroupRow = 12
	if got := table.GetCellText(logGroupRow, 0); got != "Log Group" {
		t.Fatalf("row %d = %q, want Log Group", logGroupRow, got)
	}
	var link, _ = table.GetLink(logGroupRow)
	if want := "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/orders"; link != want {
		t.Errorf("link = %q, want %q", link, want)
	}
}

func TestDetailsTable__RoleLinksToIamRole(t *testing.T) {
	const roleArn = "arn:aws:iam::123456789012:role/orders"
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{Role: aws.String(roleArn)}

	table.populateLambdaDetailsTable()

	const roleRow = 5
	if got := table.GetCellText(roleRow, 0); got != "Role" {
		t.Fatalf("row %d = %q, want Role", roleRow, got)
	}
	if link, _ := table.GetLink(roleRow); link != roleArn {
		t.Errorf("link = %q, want %q", link, roleArn)
	}
}
