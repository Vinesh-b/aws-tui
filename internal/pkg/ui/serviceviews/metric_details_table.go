package serviceviews

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
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *MetricDetailsTable) RefreshDetails(metric string, reset bool) {
	inst.currentMetric = metric
	var resultChannel = make(chan struct{})

	go func() {
		inst.data = inst.api.ListMetrics(nil, "", "", reset)
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.Box, resultChannel, func() {
		inst.populateMetricDetailsTable()
	})
}
