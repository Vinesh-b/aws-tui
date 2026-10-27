package lambda

import (
	"errors"
	"testing"
	"time"

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

func TestDetailsTable__ShowsEnvironmentAndVpcSections(t *testing.T) {
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	table.data = types.FunctionConfiguration{
		Environment: &types.EnvironmentResponse{Variables: map[string]string{
			"STAGE": "prod", "BUCKET": "orders-bucket", "LOG_LEVEL": "info",
		}},
		VpcConfig: &types.VpcConfigResponse{
			VpcId:            aws.String("vpc-1"),
			SubnetIds:        []string{"subnet-a", "subnet-b"},
			SecurityGroupIds: []string{"sg-1"},
		},
	}

	table.populateLambdaDetailsTable()

	const firstSectionRow = 11
	var want = []core.TableRow{
		{"Environment"},
		{"BUCKET", "orders-bucket"},
		{"LOG_LEVEL", "info"},
		{"STAGE", "prod"},
		{"VPC"},
		{"VPC ID", "vpc-1"},
		{"Subnet ID", "subnet-a"},
		{"Subnet ID", "subnet-b"},
		{"Security Group ID", "sg-1"},
	}
	for idx, row := range want {
		for col, text := range row {
			if got := table.GetCellText(firstSectionRow+idx, col); got != text {
				t.Errorf("cell (%d,%d) = %q, want %q", firstSectionRow+idx, col, got, text)
			}
		}
	}
	if got := table.GetCellText(firstSectionRow+len(want), 0); got != "" {
		t.Errorf("unexpected row %q after the VPC", got)
	}
}

func TestDetailsTable__LambdaOutsideVpcHasNoVpcSection(t *testing.T) {
	var table = NewLambdaDetailsTable(uitest.NewServiceContext[awsapi.LambdaApi]())
	// The api gives an empty config for lambdas that are not in a VPC
	table.data = types.FunctionConfiguration{VpcConfig: &types.VpcConfigResponse{VpcId: aws.String("")}}

	table.populateLambdaDetailsTable()

	if got := table.GetCellText(11, 0); got != "" {
		t.Errorf("unexpected row %q after the configuration", got)
	}
}

func newRunningDetailsTable(t *testing.T, api *fakeLambdaApi) *LambdaDetailsTable {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var table = NewLambdaDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return table
}

func functionWithArn(name string, arn string) types.FunctionConfiguration {
	var function = newFunction(name)
	function.FunctionArn = aws.String(arn)
	return function
}

func TestDetailsTable__TagsAreLoadedIntoTheirSection(t *testing.T) {
	var api = &fakeLambdaApi{tags: map[string]map[string]string{ordersArn: {"team": "orders", "env": "prod"}}}
	var table = newRunningDetailsTable(t, api)
	var app = table.serviceCtx.App

	uitest.OnUI(t, app, func() { table.RefreshDetails(functionWithArn("orders", ordersArn)) })

	uitest.Eventually(t, app, func() bool { return table.GetCellText(11, 0) == "Tags" })
	uitest.OnUI(t, app, func() {
		for row, want := range []core.TableRow{{"env", "prod"}, {"team", "orders"}} {
			if got := table.GetCellText(12+row, 0); got != want[0] {
				t.Errorf("tag %d = %q, want %q", row, got, want[0])
			}
		}
		if got := table.GetTitle(); got != "Lambda Details ❬orders❭" {
			t.Errorf("title = %q", got)
		}
	})
}

func TestDetailsTable__TagsOfPreviousLambdaAreDropped(t *testing.T) {
	var api = &fakeLambdaApi{
		tags: map[string]map[string]string{
			billingArn: {"team": "billing"},
			ordersArn:  {"team": "orders"},
		},
		delays: map[string]time.Duration{billingArn: 300 * time.Millisecond},
	}
	var table = newRunningDetailsTable(t, api)
	var app = table.serviceCtx.App

	uitest.OnUI(t, app, func() {
		table.RefreshDetails(functionWithArn("billing", billingArn))
		table.RefreshDetails(functionWithArn("orders", ordersArn))
	})
	uitest.Eventually(t, app, func() bool { return table.GetCellText(12, 1) == "orders" })
	uitest.Consistently(t, app, 500*time.Millisecond, func() bool { return table.GetCellText(12, 1) == "orders" },
		"a stale lambda replaced the latest one")

	uitest.OnUI(t, app, func() {
		if got := table.GetCellText(12, 1); got != "orders" {
			t.Errorf("team tag = %q, want the latest lambda's", got)
		}
		if got := table.GetTitle(); got != "Lambda Details ❬orders❭" {
			t.Errorf("title = %q, want the latest lambda's", got)
		}
	})
}

func TestDetailsTable__TagsAreNotLoadedAgainForTheSameLambda(t *testing.T) {
	var api = &fakeLambdaApi{tags: map[string]map[string]string{ordersArn: {"team": "orders"}}}
	var table = newRunningDetailsTable(t, api)
	var app = table.serviceCtx.App
	uitest.OnUI(t, app, func() { table.RefreshDetails(functionWithArn("orders", ordersArn)) })
	uitest.Eventually(t, app, func() bool { return table.GetCellText(11, 0) == "Tags" })

	uitest.OnUI(t, app, func() { table.RefreshDetails(functionWithArn("orders", ordersArn)) })

	uitest.OnUI(t, app, func() {
		if got := table.GetCellText(11, 0); got != "Tags" {
			t.Errorf("row 11 = %q, want the tags kept", got)
		}
	})
	var calls = 0
	api.mu.Lock()
	for _, call := range api.asked {
		if call == "ListTags "+ordersArn {
			calls++
		}
	}
	api.mu.Unlock()
	if calls != 1 {
		t.Errorf("ListTags calls = %d, want 1", calls)
	}
}

func TestDetailsTable__TagsFailureIsReported(t *testing.T) {
	var api = &fakeLambdaApi{tagsErr: errors.New("access denied")}
	var table = newRunningDetailsTable(t, api)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	uitest.OnUI(t, table.serviceCtx.App, func() { table.RefreshDetails(functionWithArn("orders", ordersArn)) })

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q", got)
	}
}
