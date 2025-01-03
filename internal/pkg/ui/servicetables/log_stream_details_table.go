package servicetables

import (
	"log"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/rivo/tview"
)

type LogStreamDetailsTable struct {
	*core.DetailsTable
	data   types.LogStream
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.CloudWatchLogsApi
}

func NewLogStreamDetailsTable(
	app *tview.Application,
	api *awsapi.CloudWatchLogsApi,
	logger *log.Logger,
) *LogStreamDetailsTable {
	var table = &LogStreamDetailsTable{
		DetailsTable: core.NewDetailsTable("Log Stream Details"),
		data:         types.LogStream{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	table.populateDetailsTable()

	return table
}

func (inst *LogStreamDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var timestampFormat = "2006-01-02 15:04:05.000"
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.LogStreamName)},
		{"Arn", aws.ToString(inst.data.Arn)},
		{"FirstEventTime", time.UnixMilli(
			int64(aws.ToInt64(inst.data.FirstEventTimestamp)),
		).Format(timestampFormat)},
		{"LastEventTime", time.UnixMilli(
			int64(aws.ToInt64(inst.data.LastEventTimestamp)),
		).Format(timestampFormat)},
		{"LastIngestionTime", time.UnixMilli(
			int64(aws.ToInt64(inst.data.LastIngestionTime)),
		).Format(timestampFormat)},
		{"CreatedTime", time.UnixMilli(
			int64(aws.ToInt64(inst.data.CreationTime)),
		).Format(time.DateTime)},
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LogStreamDetailsTable) RefreshDetails(logStream types.LogStream) {
	inst.data = logStream
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}
