package lambda

import (
	"context"
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

const (
	ordersArn  = "arn:aws:lambda:us-east-1:123456789012:function:orders"
	billingArn = "arn:aws:lambda:us-east-1:123456789012:function:billing"
	queueArn   = "arn:aws:sqs:us-east-1:123456789012:orders-queue"
	dlqArn     = "arn:aws:sqs:us-east-1:123456789012:orders-dlq"
)

func TestEventSource__NamesTheSource(t *testing.T) {
	var tests = []struct {
		mapping  types.EventSourceMappingConfiguration
		wantType string
		wantName string
	}{
		{types.EventSourceMappingConfiguration{EventSourceArn: aws.String(queueArn)}, "SQS", "orders-queue"},
		{types.EventSourceMappingConfiguration{
			EventSourceArn: aws.String("arn:aws:dynamodb:us-east-1:1:table/orders/stream/2024-01-02T03:04:05.000"),
		}, "DynamoDB Stream", "orders"},
		{types.EventSourceMappingConfiguration{EventSourceArn: aws.String("arn:aws:kinesis:us-east-1:1:stream/clicks")}, "Kinesis", "clicks"},
		{types.EventSourceMappingConfiguration{
			EventSourceArn: aws.String("arn:aws:kafka:us-east-1:1:cluster/events/abc-1"),
		}, "MSK", "events"},
		{types.EventSourceMappingConfiguration{
			SelfManagedEventSource: &types.SelfManagedEventSource{},
			Topics:                 []string{"orders", "refunds"},
		}, "Kafka", "orders, refunds"},
	}

	for _, tt := range tests {
		var gotType, gotName = eventSource(tt.mapping)
		if gotType != tt.wantType || gotName != tt.wantName {
			t.Errorf("%s: got %q %q, want %q %q", aws.ToString(tt.mapping.EventSourceArn), gotType, gotName, tt.wantType, tt.wantName)
		}
	}
}

func newQueueMapping() types.EventSourceMappingConfiguration {
	var modified = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.EventSourceMappingConfiguration{
		UUID:                           aws.String("uuid-1"),
		EventSourceArn:                 aws.String(queueArn),
		State:                          aws.String("Enabled"),
		LastProcessingResult:           aws.String("OK"),
		LastModified:                   &modified,
		BatchSize:                      aws.Int32(10),
		MaximumBatchingWindowInSeconds: aws.Int32(5),
		FunctionResponseTypes:          []types.FunctionResponseType{types.FunctionResponseTypeReportBatchItemFailures},
		DestinationConfig: &types.DestinationConfig{
			OnFailure: &types.OnFailure{Destination: aws.String(dlqArn)},
		},
		FilterCriteria: &types.FilterCriteria{Filters: []types.Filter{{Pattern: aws.String(`{"body":{"type":["order"]}}`)}}},
	}
}

func TestEventSourceRows__ShowsTheMappingAndLinksQueues(t *testing.T) {
	var table = core.NewDetailsTable("Trigger Details", uitest.NewAppContext())

	table.SetData(eventSourceRows(newQueueMapping()))

	uitest.AssertDetailsRows(t, table, []core.TableRow{
		{"Source", "orders-queue"},
		{"Type", "SQS"},
		{"Source ARN", queueArn},
		{"State", "Enabled"},
		{"State Reason", ""},
		{"Last Result", "OK"},
		{"Last Modified", "2024-01-02 03:04:05"},
		{"UUID", "uuid-1"},
		{"Batching"},
		{"Batch Size", "10"},
		{"Max Batching Window (s)", "5"},
		{"Starting Position", ""},
		{"Parallelization Factor", ""},
		{"Errors"},
		{"Max Retry Attempts", ""},
		{"Max Record Age (s)", ""},
		{"Split Batch On Error", ""},
		{"Report Batch Item Failures", "true"},
		{"On Failure", dlqArn},
		{"Filters"},
		{"Pattern", `{"body":{"type":["order"]}}`},
	})
	for _, tt := range []struct {
		row  int
		want string
	}{{0, queueArn}, {18, dlqArn}} {
		if link, _ := table.GetLink(tt.row); link != tt.want {
			t.Errorf("row %d link = %q, want %q", tt.row, link, tt.want)
		}
	}
}

func TestEventSourceRows__StreamSourcesAreNotLinked(t *testing.T) {
	var table = core.NewDetailsTable("Trigger Details", uitest.NewAppContext())
	var mapping = types.EventSourceMappingConfiguration{EventSourceArn: aws.String("arn:aws:kinesis:us-east-1:1:stream/clicks")}

	table.SetData(eventSourceRows(mapping))

	if link, ok := table.GetLink(0); ok {
		t.Errorf("kinesis source linked to %q, which nothing can open", link)
	}
}

func newRunningTriggersView(t *testing.T, api *fakeLambdaApi) *LambdaItemsView[eventSourceMapping] {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaTriggersView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestTriggersView__RefreshListsTriggersAndShowsSelected(t *testing.T) {
	var kinesis = types.EventSourceMappingConfiguration{
		EventSourceArn: aws.String("arn:aws:kinesis:us-east-1:1:stream/clicks"),
		State:          aws.String("Disabled"),
		BatchSize:      aws.Int32(100),
	}
	var api = &fakeLambdaApi{mappings: map[string][]types.EventSourceMappingConfiguration{
		ordersArn: {newQueueMapping(), kinesis},
	}}
	var view = newRunningTriggersView(t, api)
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	uitest.AwaitRows(t, app, view.ListTable, []core.TableRow{
		{"Source", "Type", "State", "Batch Size"},
		{"orders-queue", "SQS", "Enabled", "10"},
		{"clicks", "Kinesis", "Disabled", "100"},
	})
	uitest.OnUI(t, app, func() {
		if got := view.DetailsTable.GetCellText(0, 1); got != "orders-queue" {
			t.Errorf("details source = %q, want the first trigger", got)
		}
		view.ListTable.Select(2, 0)
		if got := view.DetailsTable.GetCellText(0, 1); got != "clicks" {
			t.Errorf("details source = %q, want the selected trigger", got)
		}
		if got := view.ListTable.GetTitle(); got != "Triggers ❬2❭ ❬orders❭" {
			t.Errorf("title = %q", got)
		}
	})
}

func TestTriggersView__NoTriggersEmptiesDetails(t *testing.T) {
	var view = newRunningTriggersView(t, &fakeLambdaApi{})
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })
	uitest.OnUI(t, app, func() {})

	uitest.Eventually(t, app, func() bool { return view.ListTable.GetTitle() == "Triggers ❬0❭ ❬orders❭" })
	uitest.OnUI(t, app, func() {
		if got := view.DetailsTable.GetCellText(0, 0); got != "" {
			t.Errorf("details row = %q, want none", got)
		}
	})
}

func TestTriggersView__SlowerEarlierLambdaIsDropped(t *testing.T) {
	var api = &fakeLambdaApi{
		mappings: map[string][]types.EventSourceMappingConfiguration{
			billingArn: {{EventSourceArn: aws.String("arn:aws:sqs:us-east-1:1:billing-queue")}},
			ordersArn:  {newQueueMapping()},
		},
		delays: map[string]time.Duration{billingArn: 300 * time.Millisecond},
	}
	var view = newRunningTriggersView(t, api)
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() {
		view.Refresh(billingArn)
		view.Refresh(ordersArn)
	})
	uitest.Eventually(t, app, func() bool { return view.ListTable.GetCellText(1, 0) == "orders-queue" })
	time.Sleep(500 * time.Millisecond)

	uitest.OnUI(t, app, func() {
		if got := view.ListTable.GetCellText(1, 0); got != "orders-queue" {
			t.Errorf("trigger = %q, want the latest lambda's", got)
		}
		if got := view.ListTable.GetTitle(); got != "Triggers ❬1❭ ❬orders❭" {
			t.Errorf("title = %q, want the latest lambda's", got)
		}
	})
}

func newRunningPermissionsView(t *testing.T, api *fakeLambdaApi) *LambdaPermissionsView {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.LambdaApi](api)
	var view = NewLambdaPermissionsView(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestPermissionsView__ShowsFormattedPolicy(t *testing.T) {
	var api = &fakeLambdaApi{policies: map[string]string{
		ordersArn: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"s3.amazonaws.com"}}]}`,
	}}
	var view = newRunningPermissionsView(t, api)
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	uitest.Eventually(t, app, func() bool { return view.GetText() != "" })
	uitest.OnUI(t, app, func() {
		var want = "{\n  \"Statement\": [\n    {\n      \"Effect\": \"Allow\",\n      \"Principal\": {\n        \"Service\": \"s3.amazonaws.com\"\n      }\n    }\n  ],\n  \"Version\": \"2012-10-17\"\n}"
		if got := view.GetText(); got != want {
			t.Errorf("policy =\n%s", got)
		}
		if !view.JsonHighlighting() {
			t.Error("policy should be highlighted")
		}
		if got := view.GetTitle(); got != "Resource-based Policy ❬orders❭" {
			t.Errorf("title = %q", got)
		}
	})
}

func TestPermissionsView__LambdaWithoutPolicySaysSo(t *testing.T) {
	var view = newRunningPermissionsView(t, &fakeLambdaApi{})
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	uitest.Eventually(t, app, func() bool { return view.GetText() == noPolicyMessage })
	uitest.OnUI(t, app, func() {
		if view.JsonHighlighting() {
			t.Error("the message should not be highlighted as json")
		}
	})
}

func TestPermissionsView__FailureIsReportedAndShowsNothing(t *testing.T) {
	var view = newRunningPermissionsView(t, &fakeLambdaApi{policyErr: errors.New("access denied")})
	var app = view.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	view.ErrorMessageCallback = callback

	uitest.OnUI(t, app, func() { view.Refresh(ordersArn) })

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q", got)
	}
	uitest.OnUI(t, app, func() {})
	uitest.OnUI(t, app, func() {
		if got := view.GetText(); got != "" {
			t.Errorf("text = %q, want nothing after a failure", got)
		}
	})
}

func TestPermissionsView__SlowerEarlierLambdaIsDropped(t *testing.T) {
	var api = &fakeLambdaApi{
		policies: map[string]string{billingArn: `{"Id":"billing"}`, ordersArn: `{"Id":"orders"}`},
		delays:   map[string]time.Duration{billingArn: 300 * time.Millisecond},
	}
	var view = newRunningPermissionsView(t, api)
	var app = view.serviceCtx.App

	uitest.OnUI(t, app, func() {
		view.Refresh(billingArn)
		view.Refresh(ordersArn)
	})
	uitest.Eventually(t, app, func() bool { return view.GetText() == "{\n  \"Id\": \"orders\"\n}" })
	time.Sleep(500 * time.Millisecond)

	uitest.OnUI(t, app, func() {
		if got := view.GetText(); got != "{\n  \"Id\": \"orders\"\n}" {
			t.Errorf("policy = %q, want the latest lambda's", got)
		}
		if got := view.GetTitle(); got != "Resource-based Policy ❬orders❭" {
			t.Errorf("title = %q, want the latest lambda's", got)
		}
	})
}

type fakeLogsApi struct {
	awsapi.CloudWatchLogsApi
}

func (f *fakeLogsApi) ListLogStreams(
	ctx context.Context, logGroupName string, searchPrefix string, reset bool,
) ([]cwlTypes.LogStream, error) {
	return nil, nil
}

func TestHomeView__PickingLambdaLoadsItsTabs(t *testing.T) {
	var function = newFunction("orders")
	function.FunctionArn = aws.String(ordersArn)
	var api = &fakeLambdaApi{
		functions: []types.FunctionConfiguration{function},
		mappings:  map[string][]types.EventSourceMappingConfiguration{ordersArn: {newQueueMapping()}},
		policies:  map[string]string{ordersArn: `{"Id":"orders"}`},
	}
	var appCtx = uitest.NewAppContext()
	var page = NewLambdaHomeView(appCtx, api, &fakeLogsApi{}, awsapi.CloudWatchMetricsApi(nil))
	uitest.ShowPage(t, appCtx.App, page)
	var opener = page.(core.ResourceOpener)
	var resource, _ = arn.Parse(ordersArn)

	// Opening a lambda picks it as Enter does
	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	uitest.Eventually(t, appCtx.App, func() bool {
		return api.wasAsked("ListEventSourceMappings", ordersArn) && api.wasAsked("GetPolicy", ordersArn) &&
			api.wasAsked("ListVersions", ordersArn) && api.wasAsked("ListAliases", ordersArn)
	})
}
