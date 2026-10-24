package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MetricDetailsView struct {
	*core.ServicePageView
	MetricListTable    *MetricListTable
	MetricDetailsTable *MetricDetailsTable
	serviceCtx         *core.ServiceContext[awsapi.CloudWatchMetricsApi]
}

func NewMetricsDetailsView(
	metricListTable *MetricListTable,
	metricDetailsTable *MetricDetailsTable,
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *MetricDetailsView {
	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var mainPage = core.NewResizableView(
		metricDetailsTable, metricsTableSize,
		metricListTable, metricsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(serviceViewCtx.AppContext)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[][]core.View{
			{metricDetailsTable},
			{metricListTable},
		},
	)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}

	metricListTable.ErrorMessageCallback = errorHandler
	metricDetailsTable.ErrorMessageCallback = errorHandler

	return &MetricDetailsView{
		ServicePageView:    serviceView,
		MetricListTable:    metricListTable,
		MetricDetailsTable: metricDetailsTable,
		serviceCtx:         serviceViewCtx,
	}
}

func (inst *MetricDetailsView) InitInputCapture() {
	inst.MetricListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.MetricDetailsTable.RefreshDetails(inst.MetricListTable.GetSelectedMetric(), false)
	})
}

func NewMetricsHomeView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer appCtx.Theme.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchMetricsApi(appCtx.Logger)
	var serviceCtx = core.NewServiceViewContext(appCtx, api)

	var metricsDetailsView = NewMetricsDetailsView(
		NewMetricsTable(serviceCtx),
		NewMetricDetailsTable(serviceCtx),
		serviceCtx,
	)
	metricsDetailsView.InitInputCapture()

	var serviceRootView = core.NewServiceRootView(string(core.CLOUDWATCH_METRICS), appCtx)

	serviceRootView.AddAndSwitchToPage("Metrics", metricsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
