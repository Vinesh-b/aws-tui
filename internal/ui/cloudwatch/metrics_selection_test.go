package cloudwatch

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestMetricsDetailsView__MovingThroughMetricsChartsOnlyTheMetricItStopsOn(t *testing.T) {
	var api = &fakeCloudWatchMetricsApi{
		metrics: []cwTypes.Metric{
			newMetric("AWS/Lambda", "Duration"), newMetric("AWS/Lambda", "Errors"),
			newMetric("AWS/Lambda", "Invocations"), newMetric("AWS/Lambda", "Throttles"),
		},
		data: newMetricData(1, 2),
	}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
	var view = NewMetricsDetailsView(
		NewMetricsTable(serviceCtx), NewMetricDetailsTable(serviceCtx), NewMetricChartView(serviceCtx), serviceCtx,
	)
	view.InitInputCapture()
	uitest.RunApp(t, serviceCtx.App)
	view.MetricListTable.RefreshMetrics(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.MetricListTable.GetTable().GetRowCount() == 5 })
	uitest.OnUI(t, serviceCtx.App, func() { view.MetricChartView.SetVisible(true) })

	var passed, last string
	uitest.OnUI(t, serviceCtx.App, func() {
		for row := 2; row <= 4; row++ {
			view.MetricListTable.Select(row, 0)
			if row == 3 {
				passed = aws.ToString(view.MetricListTable.GetSelectedMetric().MetricName)
			}
		}
		last = aws.ToString(view.MetricListTable.GetSelectedMetric().MetricName)
	})

	uitest.Eventually(t, serviceCtx.App, func() bool {
		return slices.Contains(api.chartedMetrics(), last)
	}, "the metric stopped on to be charted")
	uitest.Consistently(t, serviceCtx.App, 2*core.SelectionLoadDelay, func() bool {
		return !slices.Contains(api.chartedMetrics(), passed)
	}, "the metric moved through not to be charted")
}
