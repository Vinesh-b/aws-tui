package lambda

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudwatch"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const lambdaMetricsNamespace = "AWS/Lambda"

type lambdaMetric struct {
	name        string
	statistic   string
	description string
}

var lambdaMetrics = []lambdaMetric{
	{"Invocations", "Sum", "Times the function was invoked"},
	{"Errors", "Sum", "Invocations that resulted in a function error"},
	{"Throttles", "Sum", "Invocation requests that were throttled"},
	{"Duration", "Average", "Time spent processing an event (ms)"},
	{"ConcurrentExecutions", "Maximum", "Instances processing events at once"},
}

type LambdaMetricsTable struct {
	*core.SelectableTable[lambdaMetric]
	selectedLambda string
}

func NewLambdaMetricsTable(appCtx *core.AppContext) *LambdaMetricsTable {
	var view = &LambdaMetricsTable{
		SelectableTable: core.NewSelectableTable[lambdaMetric](
			"Metrics",
			core.TableRow{
				"Name",
				"Description",
			},
			appCtx,
		).SetColumnExpansion(0, 1),
	}

	view.SetTitleExtra("NO LAMBDA SELECTED")
	view.populateMetricsTable()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return view
}

func (inst *LambdaMetricsTable) populateMetricsTable() {
	var tableData []core.TableRow
	for _, metric := range lambdaMetrics {
		tableData = append(tableData, core.TableRow{metric.name, metric.description})
	}

	inst.SetData(tableData, lambdaMetrics, 0)
	inst.Select(1, 0)
}

func (inst *LambdaMetricsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.populateMetricsTable()
			return nil
		}

		return capture(event)
	})
}

func (inst *LambdaMetricsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		handler(row, column)
	})
}

// GetSelectedMetric returns the selected metric of the selected lambda and the
// statistic it is best charted with.
func (inst *LambdaMetricsTable) GetSelectedMetric() (types.Metric, string) {
	var row, _ = inst.GetTable().GetSelection()
	if row < 1 || len(inst.selectedLambda) == 0 {
		return types.Metric{}, ""
	}

	var metric = inst.GetPrivateData(row, 0)
	return types.Metric{
		Namespace:  aws.String(lambdaMetricsNamespace),
		MetricName: aws.String(metric.name),
		Dimensions: []types.Dimension{
			{Name: aws.String("FunctionName"), Value: aws.String(inst.selectedLambda)},
		},
	}, metric.statistic
}

func (inst *LambdaMetricsTable) SetSelectedLambda(name string) {
	inst.selectedLambda = name
	inst.SetTitleExtra(name)
	inst.RefreshTitle(0)
}

type LambdaMetricsPageView struct {
	*core.ServicePageView
	MetricsTable    *LambdaMetricsTable
	MetricChartView *cloudwatch.MetricChartView
}

func NewLambdaMetricsPageView(
	serviceCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *LambdaMetricsPageView {
	var metricsTable = NewLambdaMetricsTable(serviceCtx.AppContext)
	var metricChartView = cloudwatch.NewMetricChartView(serviceCtx)
	var serviceView = core.NewServicePageView(serviceCtx.AppContext)
	serviceView.MainPage.
		SetDirection(tview.FlexColumn).
		AddItem(metricsTable, 0, 1, true).
		AddItem(metricChartView, 0, 2, false)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	metricsTable.ErrorMessageCallback = errorHandler
	metricChartView.ErrorMessageCallback = errorHandler

	serviceView.InitViewNavigation(
		[][]core.View{
			{metricsTable, metricChartView},
		},
	)

	var view = &LambdaMetricsPageView{
		ServicePageView: serviceView,
		MetricsTable:    metricsTable,
		MetricChartView: metricChartView,
	}

	metricsTable.SetSelectionChangedFunc(func(row, column int) {
		var metric, stat = metricsTable.GetSelectedMetric()
		metricChartView.SetStatistic(stat)
		metricChartView.SetMetric(metric)
	})

	return view
}

// SetSelectedLambda leaves the chart as it is until a metric row is selected.
func (inst *LambdaMetricsPageView) SetSelectedLambda(name string) {
	inst.MetricsTable.SetSelectedLambda(name)
}

func (inst *LambdaMetricsPageView) SetVisible(visible bool) {
	inst.MetricChartView.SetVisible(visible)
}
