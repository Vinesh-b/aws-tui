package services

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	tables "aws-tui/internal/pkg/ui/servicetables"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MetricDetailsView struct {
	*core.ServicePageView
	MetricListTable    *tables.MetricListTable
	MetricDetailsTable *tables.MetricDetailsTable
	app                *tview.Application
	api                *awsapi.CloudWatchMetricsApi
}

func NewMetricsDetailsView(
	metricListTable *tables.MetricListTable,
	metricDetailsTable *tables.MetricDetailsTable,
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricDetailsView {
	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var mainPage = core.NewResizableView(
		metricDetailsTable, metricsTableSize,
		metricListTable, metricsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			metricListTable,
			metricDetailsTable,
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
		app:                app,
		api:                api,
	}
}

func (inst *MetricDetailsView) InitInputCapture() {
	inst.MetricListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.MetricDetailsTable.RefreshDetails(inst.MetricListTable.GetSeletedMetric(), false)
	})
}

func NewMetricsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) core.ServicePage {
	core.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchMetricsApi(config, logger)
	var metricsDetailsView = NewMetricsDetailsView(
		tables.NewMetricsTable(app, api, logger),
		tables.NewMetricDetailsTable(app, api, logger),
		app, api, logger,
	)
	metricsDetailsView.InitInputCapture()

	var serviceRootView = core.NewServiceRootView(app, string(CLOUDWATCH_METRICS))

	serviceRootView.AddAndSwitchToPage("Metrics", metricsDetailsView, true)

	serviceRootView.InitPageNavigation()

	return serviceRootView
}
