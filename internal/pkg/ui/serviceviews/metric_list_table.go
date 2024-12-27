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

type MetricListTable struct {
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
) *MetricListTable {
	var view = &MetricListTable{
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

func (inst *MetricListTable) populateMetricsTable() {
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

func (inst *MetricListTable) RefreshMetrics(search string, force bool) {
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

func (inst *MetricListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return inst.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshMetrics(inst.currentSearch, true)
		}

		return capture(event)
	})
}

func (inst *MetricListTable) SetSelectionChangedFunc(handler func(row int, column int)) *tview.Table {
	return inst.Table.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedMetric = inst.Table.GetCell(row, 1).Text

		handler(row, column)
	})
}

func (inst *MetricListTable) GetSeletedMetric() string {
	return inst.selectedMetric
}
