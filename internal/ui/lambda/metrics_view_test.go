package lambda

import (
	"context"
	"sync"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type metricDataRequest struct {
	metric cwTypes.Metric
	stat   string
}

type fakeCloudWatchMetricsApi struct {
	awsapi.CloudWatchMetricsApi

	mu       sync.Mutex
	requests []metricDataRequest
}

func (f *fakeCloudWatchMetricsApi) GetMetricData(
	ctx context.Context, metric cwTypes.Metric, stat string, periodSeconds int32, start time.Time, end time.Time,
) (cwTypes.MetricDataResult, error) {
	f.mu.Lock()
	f.requests = append(f.requests, metricDataRequest{metric, stat})
	f.mu.Unlock()
	return cwTypes.MetricDataResult{}, nil
}

func (f *fakeCloudWatchMetricsApi) dataRequests() []metricDataRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]metricDataRequest(nil), f.requests...)
}

func TestMetricsTable__ListsStandardMetrics(t *testing.T) {
	var view = NewLambdaMetricsTable(uitest.NewAppContext())

	uitest.AssertTableRows(t, view, []core.TableRow{
		{"Name", "Description"},
		{"Invocations", "Times the function was invoked"},
		{"Errors", "Invocations that resulted in a function error"},
		{"Throttles", "Invocation requests that were throttled"},
		{"Duration", "Time spent processing an event (ms)"},
		{"ConcurrentExecutions", "Instances processing events at once"},
	})
}

func TestMetricsTable__SelectedMetricIsScopedToLambda(t *testing.T) {
	var view = NewLambdaMetricsTable(uitest.NewAppContext())
	view.SetSelectedLambda("orders-handler")
	view.Select(4, 0)

	var metric, stat = view.GetSelectedMetric()

	if aws.ToString(metric.Namespace) != "AWS/Lambda" || aws.ToString(metric.MetricName) != "Duration" {
		t.Errorf("metric = %s/%s, want AWS/Lambda/Duration", aws.ToString(metric.Namespace), aws.ToString(metric.MetricName))
	}
	if len(metric.Dimensions) != 1 ||
		aws.ToString(metric.Dimensions[0].Name) != "FunctionName" ||
		aws.ToString(metric.Dimensions[0].Value) != "orders-handler" {
		t.Errorf("dimensions = %+v, want FunctionName=orders-handler", metric.Dimensions)
	}
	if stat != "Average" {
		t.Errorf("stat = %q, want Average", stat)
	}
}

func TestMetricsTable__NoMetricWithoutLambda(t *testing.T) {
	var view = NewLambdaMetricsTable(uitest.NewAppContext())

	var metric, _ = view.GetSelectedMetric()

	if metric.MetricName != nil {
		t.Errorf("metric = %q, want none", aws.ToString(metric.MetricName))
	}
}

func TestMetricsView__SelectingLambdaDoesNotLoadChart(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var view = NewLambdaMetricsPageView(serviceCtx)
	var app = serviceCtx.App
	uitest.RunApp(t, app)

	uitest.OnUI(t, app, func() {
		view.SetVisible(true)
		view.SetSelectedLambda("orders-handler")
	})

	uitest.OnUI(t, app, func() {})
	if got := len(api.dataRequests()); got != 0 {
		t.Errorf("requests after selecting a lambda = %d, want 0", got)
	}
}

func TestMetricsView__RowSelectionChangeLoadsMetricWithItsStatistic(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var view = NewLambdaMetricsPageView(serviceCtx)
	var app = serviceCtx.App
	uitest.RunApp(t, app)

	uitest.OnUI(t, app, func() {
		view.SetVisible(true)
		view.SetSelectedLambda("orders-handler")
		view.MetricsTable.Select(2, 0)
	})

	uitest.Eventually(t, app, func() bool { return len(api.dataRequests()) == 1 })
	var req = api.dataRequests()[0]
	if aws.ToString(req.metric.MetricName) != "Errors" || req.stat != "Sum" {
		t.Errorf("request = %s %s, want Errors Sum", aws.ToString(req.metric.MetricName), req.stat)
	}
}
