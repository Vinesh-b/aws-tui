package servicetables

import (
	"log"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/rivo/tview"
)

type LambdaVpcConfigTable struct {
	*core.DetailsTable
	data   types.FunctionConfiguration
	logger *log.Logger
	app    *tview.Application
	api    *awsapi.LambdaApi
}

func NewLambdaVpcConfigTable(
	app *tview.Application,
	api *awsapi.LambdaApi,
	logger *log.Logger,
) *LambdaVpcConfigTable {
	var table = &LambdaVpcConfigTable{
		DetailsTable: core.NewDetailsTable("VPC Config"),
		data:         types.FunctionConfiguration{},
		logger:       logger,
		app:          app,
		api:          api,
	}

	table.populateLambdaVpcConfigTable()

	return table
}

func (inst *LambdaVpcConfigTable) populateLambdaVpcConfigTable() {
	var tableData []core.TableRow
	if inst.data.VpcConfig != nil {
		var config = inst.data.VpcConfig
		tableData = []core.TableRow{
			{"VPC Id", aws.ToString(config.VpcId)},
		}
		for _, id := range config.SubnetIds {
			tableData = append(tableData, core.TableRow{
				"Subnet Id", id,
			})
		}
		for _, id := range config.SecurityGroupIds {
			tableData = append(tableData, core.TableRow{
				"Security Group Id", id,
			})
		}
	}

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaVpcConfigTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaVpcConfigTable()
	})
}
