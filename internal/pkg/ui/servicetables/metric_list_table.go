package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gdamore/tcell/v2"
)

type MetricListTable struct {
	*core.SelectableTable[types.Metric]
	selectedMetric types.Metric
	currentSearch  string
	data           []types.Metric
	filtered       []types.Metric
	serviceCtx     *core.ServiceContext[awsapi.CloudWatchMetricsApi]
}

func NewMetricsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *MetricListTable {
	var view = &MetricListTable{
		SelectableTable: core.NewSelectableTable[types.Metric](
			"Metrics",
			core.TableRow{
				"Namespace",
				"Name",
			},
			serviceViewCtx.AppContext,
		),
		selectedMetric: types.Metric{},
		currentSearch:  "",
		data:           nil,
		serviceCtx:     serviceViewCtx,
	}

	view.populateMetricsTable(view.data)
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var search = view.GetSearchText()
			view.FilterByName(search)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *MetricListTable) populateMetricsTable(data []types.Metric) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Namespace),
			aws.ToString(row.MetricName),
		})
	}

	inst.SetData(tableData, data, 0)
	inst.GetCell(0, 0).SetExpansion(1)
}

func (inst *MetricListTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(v types.Metric) string {
				return aws.ToString(v.MetricName)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricsTable(inst.filtered)
	})
}

func (inst *MetricListTable) RefreshMetrics(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListMetrics(nil, "", "", reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricsTable(inst.data)
	})
}

func (inst *MetricListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshMetrics(true)
			return nil
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
