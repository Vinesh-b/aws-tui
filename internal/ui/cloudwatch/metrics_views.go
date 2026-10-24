package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	MetricTabNameDetails = "Details"
	MetricTabNameChart   = "Chart"
)

type MetricDetailsView struct {
	*core.ServicePageView
	MetricListTable    *MetricListTable
	MetricDetailsTable *MetricDetailsTable
	MetricChartView    *MetricChartView
	serviceCtx         *core.ServiceContext[awsapi.CloudWatchMetricsApi]
}

func NewMetricsDetailsView(
	metricListTable *MetricListTable,
	metricDetailsTable *MetricDetailsTable,
	metricChartView *MetricChartView,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *MetricDetailsView {
	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var tabView = core.NewTabViewHorizontal(serviceViewCtx.AppContext).
		AddAndSwitchToTab(MetricTabNameDetails, metricDetailsTable, 0, 1, true).
		AddTab(MetricTabNameChart, metricChartView, 0, 1, true)

	tabView.SetOnTabChangeFunc(func(tabName string, index int) {
		metricChartView.SetVisible(tabName == MetricTabNameChart)
	})

	var mainPage = core.NewResizableView(
		tabView, metricsTableSize,
		metricListTable, metricsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView.GetTabDisplayView()},
			{metricListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	metricListTable.ErrorMessageCallback = errorHandler
	metricDetailsTable.ErrorMessageCallback = errorHandler
	metricChartView.ErrorMessageCallback = errorHandler

	return &MetricDetailsView{
		ServicePageView:    serviceView,
		MetricListTable:    metricListTable,
		MetricDetailsTable: metricDetailsTable,
		MetricChartView:    metricChartView,
		serviceCtx:         serviceViewCtx,
	}
}

func (inst *MetricDetailsView) InitInputCapture() {
	inst.MetricListTable.SetSelectionChangedFunc(func(row, column int) {
		var metric = inst.MetricListTable.GetSelectedMetric()
		inst.MetricDetailsTable.RefreshDetails(metric, false)
		inst.MetricChartView.SetMetric(metric)
	})
}

func NewMetricsHomeView(appCtx *core.AppContext, api awsapi.CloudWatchMetricsApi) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer appCtx.Theme.ResetGlobalStyle()

	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var metricsDetailsView = NewMetricsDetailsView(
		NewMetricsTable(serviceCtx),
		NewMetricDetailsTable(serviceCtx),
		NewMetricChartView(serviceCtx),
		serviceCtx,
	)
	metricsDetailsView.InitInputCapture()

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_METRICS), appCtx)

	serviceRootView.AddAndSwitchToPage("Metrics", metricsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
