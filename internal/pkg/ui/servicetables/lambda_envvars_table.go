package servicetables

import (
	"log"
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/rivo/tview"
)

type LambdaEnvVarsTable struct {
	*core.DetailsTable
	data   types.FunctionConfiguration
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.LambdaApi
}

func NewLambdaEnvVarsTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaEnvVarsTable {
	var table = &LambdaEnvVarsTable{
		DetailsTable: core.NewDetailsTable("Environment Variables"),
		data:         types.FunctionConfiguration{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	table.populateLambdaEnvVarsTable()

	return table
}

func (inst *LambdaEnvVarsTable) populateLambdaEnvVarsTable() {
	var tableData []core.TableRow
	if inst.data.Environment != nil {
		var envVars = inst.data.Environment.Variables
		for k, v := range envVars {
			tableData = append(tableData, core.TableRow{k, v})
		}
		sort.Slice(tableData, func(i int, j int) bool {
			return tableData[i][0] < tableData[j][0]
		})
	}

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaEnvVarsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaEnvVarsTable()
	})
}
