package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type MetricDetailsTable struct {
	*core.DetailsTable
	data       types.Metric
	serviceCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi]
}

func NewMetricDetailsTable(
	serviceViewCtx *core.ServiceContext[awsapi.CloudWatchMetricsApi],
) *MetricDetailsTable {
	var view = &MetricDetailsTable{
		DetailsTable: core.NewDetailsTable("Metric Details"),
		data:         types.Metric{},
		serviceCtx:   serviceViewCtx,
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
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateMetricDetailsTable()
	})
}
