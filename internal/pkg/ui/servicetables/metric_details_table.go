package servicetables

import (
	"log"
	"slices"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/rivo/tview"
)

type MetricDetailsTable struct {
	*core.DetailsTable
	currentMetric string
	data          *types.Metric
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
	if inst.data != nil {
		tableData = []core.TableRow{
			{"Namespace", aws.ToString(inst.data.Namespace)},
			{"MetricName", aws.ToString(inst.data.MetricName)},
			{"Dimensions", ""},
		}
		for _, dim := range inst.data.Dimensions {
			tableData = append(tableData, core.TableRow{aws.ToString(dim.Name), aws.ToString(dim.Value)})
		}
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *MetricDetailsTable) RefreshDetails(metric string, reset bool) {
	inst.currentMetric = metric
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.api.ListMetrics(nil, "", "", reset)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		var idx = slices.IndexFunc(data, func(d types.Metric) bool {
			return aws.ToString(d.MetricName) == inst.currentMetric
		})

		if idx != -1 {
			inst.data = &data[idx]
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricDetailsTable()
	})
}
