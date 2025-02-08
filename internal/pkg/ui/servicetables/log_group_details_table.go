package servicetables

import (
	"fmt"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type LogGroupDetailsTable struct {
	*core.DetailsTable
	data       types.LogGroup
	serviceCtx *core.ServiceContext[awsapi.CloudWatchLogsApi]
}

func NewLogGroupDetailsTable(
	serviceContext *core.ServiceContext[awsapi.CloudWatchLogsApi],
) *LogGroupDetailsTable {
	var table = &LogGroupDetailsTable{
		DetailsTable: core.NewDetailsTable("Log Group Details"),
		data:         types.LogGroup{},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()

	return table
}

func (inst *LogGroupDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.LogGroupName)},
		{"Arn", aws.ToString(inst.data.LogGroupArn)},
		{"KMSKey", aws.ToString(inst.data.KmsKeyId)},
		{"Class", string(inst.data.LogGroupClass)},
		{"RetentionDays", fmt.Sprintf("%d", aws.ToInt32(inst.data.RetentionInDays))},
		{"CreatedTime", time.UnixMilli(
			int64(aws.ToInt64(inst.data.CreationTime)),
		).Format(time.DateTime)},
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LogGroupDetailsTable) RefreshDetails(logGroup types.LogGroup) {
	inst.data = logGroup
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDetailsTable()
	})
}
