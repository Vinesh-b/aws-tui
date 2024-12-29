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
	*core.ServicePageView
	MetricListTable    *MetricListTable
	MetricDetailsTable *MetricDetailsTable
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

	var mainPage = core.NewResizableView(
		metricDetailsTable, metricsTableSize,
		metricListTable, metricsTableSize,
		tview.FlexRow,
	)

	var serviceView = core.NewServicePageView(app, logger)
	serviceView.AddItem(mainPage, 0, 1, true)

	serviceView.InitViewNavigation(
		[]core.View{
			metricListTable,
			metricDetailsTable,
		},
	)

	return &MetricDetailsView{
		ServicePageView:    serviceView,
		MetricListTable:    metricListTable,
		MetricDetailsTable: metricDetailsTable,
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
		AddAndSwitchToPage("Metrics", metricsDetailsView, true)

	var orderedPages = []string{
		"Metrics",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(CLOUDWATCH_METRICS), pages, orderedPages).Init()

	return serviceRootView.RootView
}
