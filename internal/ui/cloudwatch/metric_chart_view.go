package cloudwatch

import (
	"fmt"
	"strings"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
)

type chartRange struct {
	label         string
	duration      time.Duration
	periodSeconds int32
}

// Each range keeps the number of datapoints at a few hundred at most.
var chartRanges = []chartRange{
	{"1h", time.Hour, 60},
	{"3h", 3 * time.Hour, 60},
	{"12h", 12 * time.Hour, 300},
	{"1d", 24 * time.Hour, 300},
	{"3d", 72 * time.Hour, 900},
	{"1w", 7 * 24 * time.Hour, 3600},
}

var chartStatistics = []string{"Average", "Sum", "Minimum", "Maximum", "SampleCount"}

const (
	defaultChartRangeIdx = 1
	chartStatKey         = 's'
	chartRangeKey        = 't'
)

type MetricChartView struct {
	*core.TimeSeriesChart
	ErrorMessageCallback func(text string, a ...any)
	serviceCtx           *core.ServiceContext[awsapi.CloudWatchMetricsApi]
	metric               types.Metric
	statIdx              int
	rangeIdx             int
	loadedFor            string
	visible              bool
	now                  func() time.Time
}

func NewMetricChartView(
	serviceCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *MetricChartView {
	var view = &MetricChartView{
		TimeSeriesChart:      core.NewTimeSeriesChart("Metric Chart", serviceCtx.AppContext),
		ErrorMessageCallback: func(text string, a ...any) {},
		serviceCtx:           serviceCtx,
		rangeIdx:             defaultChartRangeIdx,
		now:                  time.Now,
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case chartStatKey:
			view.statIdx = (view.statIdx + 1) % len(chartStatistics)
			view.reload()
			return nil
		case chartRangeKey:
			view.rangeIdx = (view.rangeIdx + 1) % len(chartRanges)
			view.reload()
			return nil
		case core.APP_KEY_BINDINGS.Reset:
			view.reload()
			return nil
		}
		return event
	})

	view.showMessage("Select a metric")

	return view
}

func (inst *MetricChartView) title(summary string) string {
	var title = fmt.Sprintf(
		"Metric Chart: %s [%s, %s]",
		aws.ToString(inst.metric.MetricName),
		chartStatistics[inst.statIdx],
		chartRanges[inst.rangeIdx].label,
	)
	if len(summary) > 0 {
		title += " " + summary
	}
	return title
}

func (inst *MetricChartView) showMessage(message string) {
	inst.SetMessage(message)
	inst.SetTitle(inst.title(""))
}

func metricIdentity(metric types.Metric) string {
	var parts = []string{aws.ToString(metric.Namespace), aws.ToString(metric.MetricName)}
	for _, dim := range metric.Dimensions {
		parts = append(parts, aws.ToString(dim.Name)+"="+aws.ToString(dim.Value))
	}
	return strings.Join(parts, "/")
}

// SetMetric selects the metric to chart. Nothing is requested until the chart
// is visible.
func (inst *MetricChartView) SetMetric(metric types.Metric) {
	inst.metric = metric
	inst.loadedFor = ""

	if inst.visible {
		inst.load()
		return
	}
	if len(aws.ToString(metric.MetricName)) == 0 {
		inst.showMessage("Select a metric")
		return
	}
	inst.showMessage("The chart loads when this tab is opened")
}

func (inst *MetricChartView) SetVisible(visible bool) {
	inst.visible = visible
	if visible {
		inst.load()
	}
}

func (inst *MetricChartView) reload() {
	inst.loadedFor = ""
	if inst.visible {
		inst.load()
		return
	}
	inst.SetTitle(inst.title(""))
}

func (inst *MetricChartView) load() {
	if len(aws.ToString(inst.metric.MetricName)) == 0 {
		inst.showMessage("Select a metric")
		return
	}

	var stat = chartStatistics[inst.statIdx]
	var window = chartRanges[inst.rangeIdx]
	var target = fmt.Sprintf("%s|%s|%s", metricIdentity(inst.metric), stat, window.label)
	if inst.loadedFor == target {
		return
	}
	inst.loadedFor = target
	inst.showMessage("Loading...")

	var metric = inst.metric
	var end = inst.now()
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 30)
	var result types.MetricDataResult
	var loadFailed = false

	dataLoader.AsyncLoadData(func() {
		var err error
		result, err = inst.serviceCtx.Api.GetMetricData(
			inst.serviceCtx.Ctx, metric, stat, window.periodSeconds, end.Add(-window.duration), end,
		)
		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if inst.loadedFor != target {
			return
		}

		var points = chartPoints(result)
		switch {
		case loadFailed:
			inst.loadedFor = ""
			inst.showMessage("Failed to load the metric data")
		case len(points) == 0:
			inst.showMessage("No datapoints in the selected time range")
		default:
			inst.SetData(points)
			inst.SetTitle(inst.title(core.ChartSummary(points, inst.FormatValue)))
		}
	})
}

func chartPoints(result types.MetricDataResult) []core.ChartPoint {
	var count = min(len(result.Timestamps), len(result.Values))
	var points = make([]core.ChartPoint, 0, count)
	for i := range count {
		points = append(points, core.ChartPoint{Time: result.Timestamps[i], Value: result.Values[i]})
	}
	return points
}
