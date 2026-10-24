package cloudwatch

import (
	"errors"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/uitest"

	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var chartNow = time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

func newMetricData(values ...float64) cwTypes.MetricDataResult {
	var result cwTypes.MetricDataResult
	for i, v := range values {
		result.Timestamps = append(result.Timestamps, chartNow.Add(time.Duration(i-len(values))*time.Minute))
		result.Values = append(result.Values, v)
	}
	return result
}

func newChartView(t *testing.T, api *fakeCloudWatchMetricsApi) *MetricChartView {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var view = NewMetricChartView(serviceCtx)
	view.now = func() time.Time { return chartNow }
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func onChartUI(t *testing.T, view *MetricChartView, fn func()) {
	t.Helper()
	uitest.OnUI(t, view.serviceCtx.App, fn)
}

func awaitChartState(t *testing.T, view *MetricChartView, condition func() bool) {
	t.Helper()
	uitest.Eventually(t, view.serviceCtx.App, condition)
}

func pressChartKey(t *testing.T, view *MetricChartView, r rune) {
	t.Helper()
	onChartUI(t, view, func() {
		view.InputHandler()(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(tview.Primitive) {})
	})
}

func TestMetricChartView__HiddenChartRequestsNothing(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1, 2)}
	var view = newChartView(t, api)

	onChartUI(t, view, func() { view.SetMetric(newMetric("AWS/Lambda", "Invocations")) })

	if got := len(api.metricDataRequests()); got != 0 {
		t.Errorf("requests while hidden = %d, want 0", got)
	}
	onChartUI(t, view, func() {
		if !strings.Contains(view.GetMessage(), "opened") {
			t.Errorf("message = %q", view.GetMessage())
		}
	})
}

func TestMetricChartView__ShowingChartRequestsTheSelectedMetric(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1, 5, 3)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() { view.SetMetric(newMetric("AWS/Lambda", "Invocations")) })

	onChartUI(t, view, func() { view.SetVisible(true) })

	awaitChartState(t, view, func() bool { return len(view.GetPoints()) == 3 })
	var requests = api.metricDataRequests()
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	var want = metricDataRequest{"Average", 60, chartNow.Add(-3 * time.Hour), chartNow}
	if requests[0] != want {
		t.Errorf("request = %+v, want %+v", requests[0], want)
	}
	onChartUI(t, view, func() {
		var title = view.GetTitle()
		if !strings.Contains(title, "Invocations [Average, 3h]") || !strings.Contains(title, "max 5") {
			t.Errorf("title = %q", title)
		}
	})
}

func TestMetricChartView__SelectingMetricWhileVisibleLoadsImmediately(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() { view.SetVisible(true) })

	onChartUI(t, view, func() { view.SetMetric(newMetric("AWS/Lambda", "Invocations")) })

	awaitChartState(t, view, func() bool { return len(view.GetPoints()) == 1 })
}

func TestMetricChartView__ReopeningChartDoesNotRequestAgain(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() {
		view.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		view.SetVisible(true)
	})
	awaitChartState(t, view, func() bool { return len(view.GetPoints()) == 1 })

	onChartUI(t, view, func() {
		view.SetVisible(false)
		view.SetVisible(true)
	})

	if got := len(api.metricDataRequests()); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestMetricChartView__StatisticKeyCyclesAndReloads(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() {
		view.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		view.SetVisible(true)
	})
	awaitChartState(t, view, func() bool { return len(api.metricDataRequests()) == 1 })

	pressChartKey(t, view, chartStatKey)

	awaitChartState(t, view, func() bool { return len(api.metricDataRequests()) == 2 })
	if got := api.metricDataRequests()[1].stat; got != "Sum" {
		t.Errorf("statistic = %q, want Sum", got)
	}
}

func TestMetricChartView__RangeKeyCyclesWindowAndPeriod(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() {
		view.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		view.SetVisible(true)
	})
	awaitChartState(t, view, func() bool { return len(api.metricDataRequests()) == 1 })

	pressChartKey(t, view, chartRangeKey)

	awaitChartState(t, view, func() bool { return len(api.metricDataRequests()) == 2 })
	var request = api.metricDataRequests()[1]
	if request.period != 300 || !request.start.Equal(chartNow.Add(-12*time.Hour)) {
		t.Errorf("request = %+v, want a 12h window with a 300s period", request)
	}
}

func TestMetricChartView__ChangingOptionsWhileHiddenDoesNotRequest(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1)}
	var view = newChartView(t, api)
	onChartUI(t, view, func() { view.SetMetric(newMetric("AWS/Lambda", "Invocations")) })

	pressChartKey(t, view, chartStatKey)

	if got := len(api.metricDataRequests()); got != 0 {
		t.Errorf("requests while hidden = %d, want 0", got)
	}
	onChartUI(t, view, func() {
		if !strings.Contains(view.GetTitle(), "Sum") {
			t.Errorf("title = %q, want the new statistic", view.GetTitle())
		}
	})
}

func TestMetricChartView__NoMetricSelectedRequestsNothing(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{}
	var view = newChartView(t, api)

	onChartUI(t, view, func() { view.SetVisible(true) })

	if got := len(api.metricDataRequests()); got != 0 {
		t.Errorf("requests = %d, want 0", got)
	}
}

func TestMetricChartView__EmptyResultShowsMessage(t *testing.T) {
	var view = newChartView(t, &fakeCloudWatchMetricsApi{})

	onChartUI(t, view, func() {
		view.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		view.SetVisible(true)
	})

	awaitChartState(t, view, func() bool { return strings.Contains(view.GetMessage(), "No datapoints") })
}

func TestMetricChartView__ErrorIsReportedAndRetriedOnReopen(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{dataErr: errors.New("access denied")}
	var view = newChartView(t, api)
	var callback, received = uitest.CaptureErrors()
	view.ErrorMessageCallback = callback

	onChartUI(t, view, func() {
		view.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		view.SetVisible(true)
	})

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
	awaitChartState(t, view, func() bool { return strings.Contains(view.GetMessage(), "Failed") })
	onChartUI(t, view, func() {
		view.SetVisible(false)
		view.SetVisible(true)
	})
	awaitChartState(t, view, func() bool { return len(api.metricDataRequests()) == 2 })
}

func TestMetricsDetailsView__ChartTabOnlyRequestsOnceItIsOpened(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{data: newMetricData(1, 2)}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var view = NewMetricsDetailsView(
		NewMetricsTable(serviceCtx),
		NewMetricDetailsTable(serviceCtx),
		NewMetricChartView(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	uitest.OnUI(t, serviceCtx.App, func() {
		view.MetricChartView.SetMetric(newMetric("AWS/Lambda", "Invocations"))
		serviceCtx.App.SetFocus(view.MetricDetailsTable)
	})
	if got := len(api.metricDataRequests()); got != 0 {
		t.Fatalf("requested before the tab was opened: %d", got)
	}

	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)

	uitest.Eventually(t, serviceCtx.App, func() bool { return len(view.MetricChartView.GetPoints()) == 2 })
	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)
	uitest.PressKey(t, serviceCtx.App, tcell.KeyTab)
	if got := len(api.metricDataRequests()); got != 1 {
		t.Errorf("requests after reopening = %d, want 1", got)
	}
}

func TestChartPoints__PairsTimestampsWithValues(t *testing.T) {
	var points = chartPoints(newMetricData(4, 7))

	if len(points) != 2 || points[0].Value != 4 || points[1].Value != 7 || !points[0].Time.Before(points[1].Time) {
		t.Errorf("points = %+v", points)
	}
	var uneven = cwTypes.MetricDataResult{Timestamps: []time.Time{chartNow, chartNow}, Values: []float64{1}}
	if got := len(chartPoints(uneven)); got != 1 {
		t.Errorf("uneven result gave %d points, want 1", got)
	}
}
