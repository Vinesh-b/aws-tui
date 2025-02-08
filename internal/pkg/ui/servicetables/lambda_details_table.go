package servicetables

import (
	"fmt"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaDetailsTable struct {
	*core.DetailsTable
	data       types.FunctionConfiguration
	serviceCtx *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		DetailsTable: core.NewDetailsTable("Lambda Details"),
		data:         types.FunctionConfiguration{},
		serviceCtx:   serviceCtx,
	}

	table.populateLambdaDetailsTable()

	return table
}

func (inst *LambdaDetailsTable) populateLambdaDetailsTable() {
	var tableData []core.TableRow
	var loggingConfig = inst.data.LoggingConfig
	var logGroup = ""
	var sysLogLevel = ""
	if loggingConfig != nil {
		logGroup = aws.ToString(loggingConfig.LogGroup)
		sysLogLevel = string(loggingConfig.SystemLogLevel)

	}

	tableData = []core.TableRow{
		{"Description", aws.ToString(inst.data.Description)},
		{"Arn", aws.ToString(inst.data.FunctionArn)},
		{"Version", aws.ToString(inst.data.Version)},
		{"MemorySize", fmt.Sprintf("%d", aws.ToInt32(inst.data.MemorySize))},
		{"Runtime", string(inst.data.Runtime)},
		{"Arch", fmt.Sprintf("%v", inst.data.Architectures)},
		{"Timeout", fmt.Sprintf("%d", aws.ToInt32(inst.data.Timeout))},
		{"LoggingGroup", logGroup},
		{"SystemLogLevel", sysLogLevel},
		{"State", string(inst.data.State)},
		{"LastModified", aws.ToString(inst.data.LastModified)},
		{"Role", aws.ToString(inst.data.Role)},
	}

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaDetailsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaDetailsTable()
	})
}
