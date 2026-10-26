package cloudwatch

import (
	"fmt"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

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
		DetailsTable: core.NewDetailsTable("Log Group Details", serviceContext.AppContext),
		data:         types.LogGroup{},
		serviceCtx:   serviceContext,
	}

	table.populateDetailsTable()

	return table
}

// retention is unset for groups whose events never expire
func (inst *LogGroupDetailsTable) retention() string {
	if inst.data.RetentionInDays == nil {
		return "Never expire"
	}
	return fmt.Sprintf("%d", *inst.data.RetentionInDays)
}

func (inst *LogGroupDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.LogGroupName)},
		{"ARN", aws.ToString(inst.data.LogGroupArn)},
		{"KMS Key", aws.ToString(inst.data.KmsKeyId)},
		{"Class", string(inst.data.LogGroupClass)},
		{"Retention (days)", inst.retention()},
		{"Created", core.FormatUnixMilli(inst.data.CreationTime)},
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
