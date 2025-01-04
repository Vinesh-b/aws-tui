package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/rivo/tview"
)

type MetricDetailsTable struct {
	*core.DetailsTable
	data   types.Metric
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.CloudWatchMetricsApi
}

func NewMetricDetailsTable(
	app *tview.Application,
	api *awsapi.CloudWatchMetricsApi,
	logger *log.Logger,
) *MetricDetailsTable {
	var view = &MetricDetailsTable{
		DetailsTable: core.NewDetailsTable("Metric Details"),
		data:         types.Metric{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	return view
}

func (inst *MetricDetailsTable) populateMetricDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Namespace", aws.ToString(inst.data.Namespace)},
		{"MetricName", aws.ToString(inst.data.MetricName)},
		{"Dimensions", ""},
	}
	for _, dim := range inst.data.Dimensions {
		tableData = append(tableData,
			core.TableRow{
				aws.ToString(dim.Name),
				aws.ToString(dim.Value),
			},
		)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *MetricDetailsTable) RefreshDetails(metric types.Metric, reset bool) {
	inst.data = metric
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricDetailsTable()
	})
}
