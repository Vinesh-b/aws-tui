package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MetricDetailsView struct {
	MetricListTable    *MetricListTable
	MetricDetailsTable *MetricDetailsTable
	RootView           *tview.Flex
	app                *tview.Application
	api                *awsapi.CloudWatchMetricsApi
}

func NewMetricsDetailsView(
	metricListTable *MetricListTable,
	metricDetailsTable *MetricDetailsTable,
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricDetailsView {
	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricDetailsTable.RootView, 0, metricsTableSize, false).
		AddItem(metricListTable.RootView, 0, metricsTableSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		metricDetailsTable.RootView, metricListTable.RootView,
		detailsTableSize, metricsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			metricListTable.RootView,
			metricDetailsTable.RootView,
		},
	)

	return &MetricDetailsView{
		MetricListTable:    metricListTable,
		MetricDetailsTable: metricDetailsTable,
		RootView:           serviceView.RootView,
		app:                app,
		api:                api,
	}
}

func (inst *MetricDetailsView) InitInputCapture() {
	inst.MetricListTable.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.MetricListTable.RefreshMetrics(inst.MetricListTable.GetSearchText(), false)
		}
	})

	inst.MetricListTable.SetSelectionChangedFunc(func(row, column int) {
		inst.MetricDetailsTable.RefreshDetails(inst.MetricListTable.GetSeletedMetric(), false)
	})
}

func NewMetricsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x660000))
	defer core.ResetGlobalStyle()

	var api = awsapi.NewCloudWatchMetricsApi(config, logger)
	var metricsDetailsView = NewMetricsDetailsView(
		NewMetricsTable(app, api, logger),
		NewMetricDetailsTable(app, api, logger),
		app, api, logger,
	)
	metricsDetailsView.InitInputCapture()

	var pages = tview.NewPages().
		AddAndSwitchToPage("Metrics", metricsDetailsView.RootView, true)

	var orderedPages = []string{
		"Metrics",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_METRICS), pages, orderedPages).Init()

	return serviceRootView.RootView
}
