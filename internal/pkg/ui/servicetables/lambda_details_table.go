package servicetables

import (
	"fmt"
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/rivo/tview"
)

type LambdaDetailsTable struct {
	*core.DetailsTable
	data   types.FunctionConfiguration
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.LambdaApi
}

func NewLambdaDetailsTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaDetailsTable {
	var table = &LambdaDetailsTable{
		DetailsTable: core.NewDetailsTable("Lambda Details"),
		data:         types.FunctionConfiguration{},
		logger:       logger,
		app:          app,
		api:          api,
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

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaDetailsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaDetailsTable()
	})
}
