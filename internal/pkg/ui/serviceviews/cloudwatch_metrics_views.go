package serviceviews

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MetricsTable struct {
	*core.SelectableTable[any]
	selectedMetric string
	currentSearch  string
	data           map[string]types.Metric
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.CloudWatchMetricsApi
}

func NewMetricsTable(
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricsTable {
	var view = &MetricsTable{
		SelectableTable: core.NewSelectableTable[any](
			"Metrics",
			core.TableRow{
				"Namespace",
				"Name",
			},
		),
		selectedMetric: "",
		currentSearch:  "",
		data:           nil,
		logger:         logger,
		app:            app,
		api:            api,
	}

	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.populateMetricsTable()

	return view
}

func (inst *MetricsTable) populateMetricsTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Namespace),
			aws.ToString(row.MetricName),
		})
	}

	inst.SetData(tableData)
	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *MetricsTable) SetSelectionChangedFunc(handler func(row int, column int)) *tview.Table {
	return inst.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedMetric = inst.Table.GetCell(row, 1).Text

		handler(row, column)
	})
}

func (inst *MetricsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return inst.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshMetrics(inst.currentSearch, true)
		}

		return capture(event)
	})
}

func (inst *MetricsTable) GetSeletedMetric() string {
	return inst.selectedMetric
}

func (inst *MetricsTable) RefreshMetrics(search string, force bool) {
	inst.currentSearch = search
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			inst.data = inst.api.ListMetrics(nil, "", "", force)
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateMetricsTable()
	})
}

type MetricDetailsTable struct {
	*core.DetailsTable
	currentMetric string
	data          map[string]types.Metric
	logger        *log.Logger
	app           *tview.Application
	api           *awsapi.CloudWatchMetricsApi
}

func NewMetricDetailsTable(
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricDetailsTable {
	var view = &MetricDetailsTable{
		DetailsTable:  core.NewDetailsTable("Metric Details"),
		currentMetric: "",
		data:          nil,
		logger:        logger,
		app:           app,
		api:           api,
	}

	return view
}

func (inst *MetricDetailsTable) populateMetricDetailsTable() {
	var tableData []core.TableRow
	var detail, found = inst.data[inst.currentMetric]
	if found {
		tableData = []core.TableRow{
			{"Namespace", aws.ToString(detail.Namespace)},
			{"MetricName", aws.ToString(detail.MetricName)},
			{"Dimensions", ""},
		}
		for _, dim := range detail.Dimensions {
			tableData = append(tableData, core.TableRow{aws.ToString(dim.Name), aws.ToString(dim.Value)})
		}
	}

	inst.SetData(tableData)
	inst.Table.Select(0, 0)
	inst.Table.ScrollToBeginning()
}

func (inst *MetricDetailsTable) RefreshDetails(metric string, force bool) {
	inst.currentMetric = metric
	var resultChannel = make(chan struct{})

	go func() {
		inst.data = inst.api.ListMetrics(nil, "", "", force)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Table.Box, resultChannel, func() {
		inst.populateMetricDetailsTable()
	})
}

type MetricDetailsView struct {
	MetricsTable       *MetricsTable
	MetricDetailsTable *MetricDetailsTable
	RootView           *tview.Flex
	searchableView     *core.SearchableView
	app                *tview.Application
	api                *awsapi.CloudWatchMetricsApi
}

func NewMetricsDetailsView(
	metricsTable *MetricsTable,
	metricDetailsTable *MetricDetailsTable,
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricDetailsView {
	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricDetailsTable.Table, 0, metricsTableSize, false).
		AddItem(metricsTable.Table, 0, metricsTableSize, true)

	var serviceView = core.NewServiceView(app, logger, mainPage)

	serviceView.SetResizableViews(
		metricDetailsTable.Table, metricsTable.Table,
		detailsTableSize, metricsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			metricsTable.Table,
			metricDetailsTable.Table,
		},
	)

	return &MetricDetailsView{
		MetricsTable:       metricsTable,
		MetricDetailsTable: metricDetailsTable,
		RootView:           serviceView.RootView,
		searchableView:     serviceView.SearchableView,
		app:                app,
		api:                api,
	}
}

func (inst *MetricDetailsView) InitInputCapture() {
	inst.MetricsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.MetricsTable.RefreshMetrics(inst.searchableView.GetText(), true)
		}
		return event
	})

	inst.MetricsTable.SetSelectionChangedFunc(func(row, column int) {
		inst.MetricDetailsTable.RefreshDetails(inst.MetricsTable.GetSeletedMetric(), false)
	})

	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.MetricsTable.RefreshMetrics(inst.searchableView.GetText(), false)
		}
	})
}

func CreateMetricsHomeView(
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
		app, api, logger)
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
