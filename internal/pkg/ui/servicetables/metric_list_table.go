package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MetricListTable struct {
	*core.SelectableTable[types.Metric]
	selectedMetric types.Metric
	currentSearch  string
	data           []types.Metric
	logger         *log.Logger
	app            *tview.Application
	api            *awsapi.CloudWatchMetricsApi
}

func NewMetricsTable(
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricListTable {
	var view = &MetricListTable{
		SelectableTable: core.NewSelectableTable[types.Metric](
			"Metrics",
			core.TableRow{
				"Namespace",
				"Name",
			},
		),
		selectedMetric: types.Metric{},
		currentSearch:  "",
		data:           nil,
		logger:         logger,
		app:            app,
		api:            api,
	}

	view.populateMetricsTable()
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.RefreshMetrics(false)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.RefreshMetrics(false)
	})

	return view
}

func (inst *MetricListTable) populateMetricsTable() {
	var tableData []core.TableRow
	var privateData []types.Metric
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Namespace),
			aws.ToString(row.MetricName),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *MetricListTable) RefreshMetrics(force bool) {
	var search = inst.GetSearchText()
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		if len(search) > 0 {
			inst.data = inst.api.FilterByName(search)
		} else {
			var err error = nil
			inst.data, err = inst.api.ListMetrics(nil, "", "", force)
			if err != nil {
				inst.ErrorMessageCallback(err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricsTable()
	})
}

func (inst *MetricListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshMetrics(true)
		}

		return capture(event)
	})
}

func (inst *MetricListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedMetric = inst.GetPrivateData(row, 0)

		handler(row, column)
	})
}

func (inst *MetricListTable) GetSeletedMetric() types.Metric {
	return inst.selectedMetric
}
