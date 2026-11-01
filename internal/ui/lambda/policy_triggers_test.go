package lambda

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

const ordersApiArn = "arn:aws:execute-api:us-east-1:123456789012:abc123/prod/GET/orders/{id}"

// ordersPolicy lets API Gateway call the lambda from two methods, and S3 call
// it from a bucket
const ordersPolicy = `{"Version":"2012-10-17","Statement":[
	{"Sid":"s3","Effect":"Allow","Principal":{"Service":"s3.amazonaws.com"},"Action":"lambda:InvokeFunction",
	 "Condition":{"ArnLike":{"AWS:SourceArn":"arn:aws:s3:::uploads"}}},
	{"Sid":"post","Effect":"Allow","Principal":{"Service":"apigateway.amazonaws.com"},"Action":"lambda:InvokeFunction",
	 "Condition":{"ArnLike":{"aws:SourceArn":"arn:aws:execute-api:us-east-1:123456789012:xyz789/*/POST/orders"}}},
	{"Sid":"get","Effect":"Allow","Principal":{"Service":["apigateway.amazonaws.com"]},"Action":"lambda:InvokeFunction",
	 "Condition":{"ArnLike":{"AWS:SourceArn":"` + ordersApiArn + `"},
	              "StringEquals":{"AWS:SourceAccount":"123456789012"}}}
]}`

func TestApiGatewayTriggers__AreTheStatementsAllowingApiGateway(t *testing.T) {
	var triggers = apiGatewayTriggers(ordersPolicy)

	var names []string
	for _, trigger := range triggers {
		names = append(names, trigger.Sid+"="+trigger.name())
	}
	if got := strings.Join(names, ","); got != "get=abc123 GET /orders/{id},post=xyz789 POST /orders" {
		t.Errorf("triggers = %s", got)
	}
	if triggers[0].SourceAccount != "123456789012" {
		t.Errorf("source account = %q", triggers[0].SourceAccount)
	}
}

func TestApiGatewayTriggers__ReadsEveryShapeOfPolicy(t *testing.T) {
	for name, tc := range map[string]struct {
		policy string
		want   string
	}{
		"no policy":   {"", ""},
		"not json":    {"{", ""},
		"denied":      {`{"Statement":[{"Effect":"Deny","Principal":{"Service":"apigateway.amazonaws.com"}}]}`, ""},
		"any account": {`{"Statement":[{"Effect":"Allow","Principal":"*"}]}`, ""},
		"one statement": {`{"Statement":{"Effect":"Allow","Principal":{"Service":"apigateway.amazonaws.com"},` +
			`"Condition":{"ArnEquals":{"AWS:SourceArn":"arn:aws:execute-api:us-east-1:1:abc123/*/$default"}}}}`,
			"abc123 $default"},
		"any api": {`{"Statement":[{"Effect":"Allow","Principal":{"Service":"apigateway.amazonaws.com"}}]}`, "Any API"},
	} {
		t.Run(name, func(t *testing.T) {
			var names []string
			for _, trigger := range apiGatewayTriggers(tc.policy) {
				names = append(names, trigger.name())
			}
			if got := strings.Join(names, ","); got != tc.want {
				t.Errorf("triggers = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApiGatewayTriggerRows__ShowsTheMethod(t *testing.T) {
	var table = core.NewDetailsTable("Trigger Details", uitest.NewAppContext())

	table.SetData(apiGatewayTriggerRows(apiGatewayTriggers(ordersPolicy)[0]))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Source", "abc123 GET /orders/{id}"},
		{"Type", "API Gateway"},
		{"Source ARN", ordersApiArn},
		{"API ID", "abc123"},
		{"Stage", "prod"},
		{"Method", "GET"},
		{"Path", "/orders/{id}"},
		{"Source Account", "123456789012"},
		{"Statement ID", "get"},
	})
}

func TestLambdaTrigger__RawDataIsWhatItWasReadFrom(t *testing.T) {
	var trigger = lambdaTrigger{apiGateway: &apiGatewayTriggers(ordersPolicy)[0]}
	var mapping = newQueueMapping()

	if got, _ := trigger.MarshalJSON(); !strings.Contains(string(got), `"Sid":"get"`) {
		t.Errorf("raw = %s, want the policy statement", got)
	}
	if got, _ := (lambdaTrigger{mapping: &mapping}).MarshalJSON(); !strings.Contains(string(got), queueArn) {
		t.Errorf("raw = %s, want the mapping", got)
	}
}

func TestTriggersView__ListsApiGatewayTriggersAfterTheMappings(t *testing.T) {
	var api = &fakeLambdaApi{
		mappings: map[string][]types.EventSourceMappingConfiguration{ordersArn: {newQueueMapping()}},
		policies: map[string]string{ordersArn: ordersPolicy},
	}
	var view = newRunningTriggersView(t, api)
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	uitest.AwaitRows(t, app, view.ListTable, []core.TableRow{
		{"Source", "Type", "State", "Batch Size"},
		{"orders-queue", "SQS", "Enabled", "10"},
		{"abc123 GET /orders/{id}", "API Gateway", "", ""},
		{"xyz789 POST /orders", "API Gateway", "", ""},
	})
	uitest.OnUI(t, app, func() {
		view.ListTable.Select(2, 0)
		if got := view.DetailsTable.GetCellText(1, 1); got != "API Gateway" {
			t.Errorf("details type = %q, want the selected API Gateway trigger", got)
		}
	})
}

func TestTriggersView__MappingsAreShownWhenThePolicyFails(t *testing.T) {
	var api = &fakeLambdaApi{
		mappings:  map[string][]types.EventSourceMappingConfiguration{ordersArn: {newQueueMapping()}},
		policyErr: errors.New("access denied"),
	}
	var view = newRunningTriggersView(t, api)
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.SetErrorMessageCallback(callback)

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q", got)
	}
	uitest.AwaitRows(t, app, view.ListTable, []core.TableRow{
		{"Source", "Type", "State", "Batch Size"},
		{"orders-queue", "SQS", "Enabled", "10"},
	})
}

func TestTriggersView__FocusKeysMoveBetweenTheListAndTheDetails(t *testing.T) {
	var api = &fakeLambdaApi{mappings: map[string][]types.EventSourceMappingConfiguration{ordersArn: {newQueueMapping()}}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaTriggersView(serviceCtx)
	var below = core.NewSelectableTable[string]("Lambdas", core.TableRow{"Name"}, serviceCtx.AppContext)

	// The tab is laid out as on the lambda page, above the list of lambdas
	var tabView = core.NewTabViewHorizontal(serviceCtx.AppContext).
		AddAndSwitchToTab(LambdaTabTriggers, view, 0, 1, true)
	var page = core.NewServicePageView(serviceCtx.AppContext)
	page.MainPage.AddItem(tabView, 0, 1, true).AddItem(below, 0, 1, false)
	page.InitViewNavigation([][]core.View{{tabView}, {below}})

	var app = serviceCtx.App
	uitest.ShowPage(t, app, page)
	uitest.OnUI(t, app, func() {
		view.Refresh(ordersArn)
		app.SetFocus(view.ListTable)
	})

	uitest.PressKey(t, app, core.APP_KEY_BINDINGS.ViewFocusRight)
	uitest.OnUI(t, app, func() {
		if !view.DetailsTable.HasFocus() {
			t.Error("right did not move focus to the trigger details")
		}
	})

	uitest.PressKey(t, app, core.APP_KEY_BINDINGS.ViewFocusLeft)
	uitest.OnUI(t, app, func() {
		if !view.ListTable.HasFocus() {
			t.Error("left did not move focus back to the triggers")
		}
	})

	uitest.PressKey(t, app, core.APP_KEY_BINDINGS.ViewFocusDown)
	uitest.OnUI(t, app, func() {
		if !below.HasFocus() {
			t.Error("down did not move focus out of the tab")
		}
	})
}
