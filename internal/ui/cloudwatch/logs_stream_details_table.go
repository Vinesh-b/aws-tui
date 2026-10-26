package cloudwatch

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type LogStreamDetailsTable struct {
	*core.DetailsTable
	data       types.LogStream
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogStreamDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogStreamDetailsTable {
	var table = &LogStreamDetailsTable{
		DetailsTable: core.NewDetailsTable("Log Stream Details", serviceContext.AppContext),
		data:         types.LogStream{},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()

	return table
}

func (inst *LogStreamDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.LogStreamName)},
		{"ARN", aws.ToString(inst.data.Arn)},
		{"First Event", core.FormatUnixMilli(inst.data.FirstEventTimestamp)},
		{"Last Event", core.FormatUnixMilli(inst.data.LastEventTimestamp)},
		{"Last Ingestion", core.FormatUnixMilli(inst.data.LastIngestionTime)},
		{"Created", core.FormatUnixMilli(inst.data.CreationTime)},
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LogStreamDetailsTable) RefreshDetails(logStream types.LogStream) {
	inst.data = logStream
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}
