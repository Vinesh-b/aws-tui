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

func populateMetricsTable(table *tview.Table, data []types.Metric) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Namespace),
			aws.ToString(row.MetricName),
		})
	}

	core.InitSelectableTable(table, "Metrics",
		core.TableRow{
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
	var tableData []core.TableRow
	if data != nil {
		tableData = []core.TableRow{
			{"Namespace", aws.ToString(data.Namespace)},
			{"MetricName", aws.ToString(data.MetricName)},
			{"Dimensions", ""},
		}
		for _, dim := range data.Dimensions {
			tableData = append(tableData, core.TableRow{aws.ToString(dim.Name), aws.ToString(dim.Value)})
		}
	}

	core.InitBasicTable(table, "Metric Details", tableData, false)
	table.Select(0, 0)
	table.ScrollToBeginning()
}

type MetricsDetailsView struct {
	MetricsTable   *tview.Table
	HistoryTable   *tview.Table
	DetailsTable   *tview.Table
	RootView       *tview.Flex
	searchableView *core.SearchableView
	app            *tview.Application
	api            *awsapi.CloudWatchMetricsApi
}

func NewMetricsDetailsView(
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricsDetailsView {
	var metricsTable = tview.NewTable()
	populateMetricsTable(metricsTable, make([]types.Metric, 0))

	var detailsTable = tview.NewTable()
	populateMetricDetailsTable(detailsTable, nil)

	const metricsTableSize = 3500
	const detailsTableSize = 3500

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(detailsTable, 0, metricsTableSize, false).
		AddItem(metricsTable, 0, metricsTableSize, true)

	var searchabelView = core.NewSearchableView(app, logger, mainPage)
	var serviceView = core.NewServiceView(app, logger)

	serviceView.RootView = searchabelView.RootView

	serviceView.SetResizableViews(
		detailsTable, metricsTable,
		detailsTableSize, metricsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			metricsTable,
			detailsTable,
		},
	)

	return &MetricsDetailsView{
		MetricsTable:   metricsTable,
		DetailsTable:   detailsTable,
		RootView:       serviceView.RootView,
		searchableView: searchabelView,
		app:            app,
		api:            api,
	}
}

func (inst *MetricsDetailsView) RefreshMetrics(search string, force bool) {
	var data []types.Metric
	var resultChannel = make(chan struct{})

	go func() {
		data = inst.api.ListMetrics(nil, "", "", force)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.MetricsTable.Box, resultChannel, func() {
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

	go core.LoadData(inst.app, inst.MetricsTable.Box, resultChannel, func() {
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
			inst.RefreshMetrics(inst.searchableView.GetText(), true)
		}
		return event
	})

	inst.MetricsTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.RefreshDetails()
	})

	inst.searchableView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshMetrics(inst.searchableView.GetText(), false)
			inst.app.SetFocus(inst.MetricsTable)
		case tcell.KeyEsc:
			inst.searchableView.SetText("")
		default:
			return
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
	var metricsDetailsView = NewMetricsDetailsView(app, api, logger)
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
