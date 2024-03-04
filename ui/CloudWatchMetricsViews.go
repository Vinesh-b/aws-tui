package ui

import (
	"log"

	"aws-tui/cloudwatch"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func populateMetricsTable(table *tview.Table, data []types.Metric) {
	var tableData []tableRow
	for _, row := range data {
		tableData = append(tableData, tableRow{
			aws.ToString(row.Namespace),
			aws.ToString(row.MetricName),
		})
	}

	initSelectableTable(table, "Metrics",
		tableRow{
			"Namespace",
			"Name",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(1, 0)
}

func populateMetricDetailsTable(table *tview.Table, data *types.Metric) {
	var tableData []tableRow
	if data != nil {
		tableData = []tableRow{
			{"Namespace", aws.ToString(data.Namespace)},
			{"MetricName", aws.ToString(data.MetricName)},
			{"Dimensions", ""},
		}
		for _, dim := range data.Dimensions {
			tableData = append(tableData, tableRow{aws.ToString(dim.Name), aws.ToString(dim.Value)})
		}
	}

	initBasicTable(table, "Metric Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

type MetricsDetailsView struct {
	MetricsTable *tview.Table
	HistoryTable *tview.Table
	DetailsTable *tview.Table
	SearchInput  *tview.InputField
	RootView     *tview.Flex
	app          *tview.Application
	api          *cloudwatch.CloudWatchMetricsApi
}

func NewMetricsDetailsView(
	app *tview.Application,
	api *cloudwatch.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricsDetailsView {
	var metricsTable = tview.NewTable()
	populateMetricsTable(metricsTable, make([]types.Metric, 0))

	var detailsTable = tview.NewTable()
	populateMetricDetailsTable(detailsTable, nil)

	var inputField = createSearchInput("Metrics")

	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var serviceView = NewServiceView(app)
	serviceView.RootView.
		AddItem(detailsTable, 0, metricsTableSize, false).
		AddItem(metricsTable, 0, metricsTableSize, false).
		AddItem(tview.NewFlex().
			AddItem(inputField, 0, 1, true),
			3, 0, true,
		)

	serviceView.SetResizableViews(
		detailsTable, metricsTable,
		detailsTableSize, metricsTableSize,
	)

	serviceView.InitViewNavigation(
		[]view{
			inputField,
			metricsTable,
			detailsTable,
		},
	)

	return &MetricsDetailsView{
		MetricsTable: metricsTable,
		DetailsTable: detailsTable,
		SearchInput:  inputField,
		RootView:     serviceView.RootView,
		app:          app,
		api:          api,
	}

}

func (inst *MetricsDetailsView) RefreshMetrics(search string, force bool) {
	var data []types.Metric
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListMetrics(nil, "", "", force)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.MetricsTable.Box, resultChannel, func() {
		populateMetricsTable(inst.MetricsTable, data)
	})
}

func (inst *MetricsDetailsView) RefreshDetails() {
	var data []types.Metric
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListMetrics(nil, "", "", false)
		resultChannel <- struct{}{}
	}()

	go loadData(inst.app, inst.MetricsTable.Box, resultChannel, func() {
		var row, _ = inst.MetricsTable.GetSelection()
		if row < len(data) {

			populateMetricDetailsTable(inst.DetailsTable, &data[row])
		}
	})
}

func (inst *MetricsDetailsView) InitInputCapture() {

	inst.MetricsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshMetrics(inst.SearchInput.GetText(), true)
		}
		return event
	})

	inst.MetricsTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.RefreshDetails()
	})

	inst.SearchInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshMetrics(inst.SearchInput.GetText(), false)
			inst.app.SetFocus(inst.MetricsTable)
		case tcell.KeyEsc:
			inst.SearchInput.SetText("")
		default:
			return
		}
	})
}

func createMetricsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	changeColourScheme(tcell.NewHexColor(0x660000))
	defer resetGlobalStyle()

	var api = cloudwatch.NewCloudWatchMetricsApi(config, logger)
	var metricsDetailsView = NewMetricsDetailsView(app, api, logger)
	metricsDetailsView.InitInputCapture()

	var pages = tview.NewPages().
		AddAndSwitchToPage("Metrics", metricsDetailsView.RootView, true)

	var orderedPages = []string{
		"Metrics",
	}

	var serviceRootView = NewServiceRootView(
		app, string(CLOUDWATCH_METRICS), pages, orderedPages).Init()

	return serviceRootView.RootView
}
