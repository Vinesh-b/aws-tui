package cloudwatch

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch/cloudwatchfake"
	"aws-tui/internal/ui/uitest"

	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestMetricsDetailsView__MovingThroughMetricsChartsOnlyTheMetricItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = cloudwatchfake.NewMetrics(
			cloudwatchfake.WithMetrics([]cwTypes.Metric{
				newMetric("AWS/Lambda", "Duration"), newMetric("AWS/Lambda", "Errors"),
				newMetric("AWS/Lambda", "Invocations"), newMetric("AWS/Lambda", "Throttles"),
			}),
			cloudwatchfake.WithMetricData(newMetricData(1, 2)),
		)
		var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudWatchMetricsApi](api)
		var view = NewMetricsDetailsView(
			NewMetricsTable(serviceCtx), NewMetricDetailsTable(serviceCtx), NewMetricChartView(serviceCtx), serviceCtx,
		)
		view.InitInputCapture()
		uitest.RunApp(t, serviceCtx.App)
		view.MetricListTable.RefreshMetrics(true)
		uitest.Eventually(t, serviceCtx.App, func() bool { return view.MetricListTable.GetTable().GetRowCount() == 5 })
		uitest.OnUI(t, serviceCtx.App, func() { view.MetricChartView.SetVisible(true) })

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.MetricListTable, Passed: "Invocations", StopsOn: "Throttles", Loaded: api.Charted,
		})
	})
}
