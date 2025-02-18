package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaVpcConfigTable struct {
	*core.DetailsTable
	data       types.FunctionConfiguration
	serviceCtx *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaVpcConfigTable(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaVpcConfigTable {
	var table = &LambdaVpcConfigTable{
		DetailsTable: core.NewDetailsTable("VPC Config", serviceCtx.AppContext),
		data:         types.FunctionConfiguration{},
		serviceCtx:   serviceCtx,
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

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaVpcConfigTable()
	})
}
