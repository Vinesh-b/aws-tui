package servicetables

import (
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type LambdaTagsTable struct {
	*core.DetailsTable
	data       types.FunctionConfiguration
	tags       map[string]string
	serviceCtx *core.ServiceContext[awsapi.LambdaApi]
}

func NewLambdaTagsTable(
	serviceCtx *core.ServiceContext[awsapi.LambdaApi],
) *LambdaTagsTable {
	var table = &LambdaTagsTable{
		DetailsTable: core.NewDetailsTable("Tags", serviceCtx.AppContext),
		data:         types.FunctionConfiguration{},
		serviceCtx:   serviceCtx,
	}

	table.populateLambdaTagsTable()

	return table
}

func (inst *LambdaTagsTable) populateLambdaTagsTable() {
	var tableData []core.TableRow
	for k, v := range inst.tags {
		tableData = append(tableData, core.TableRow{k, v})
	}

	sort.Slice(tableData, func(i int, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetTitleExtra(aws.ToString(inst.data.FunctionName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *LambdaTagsTable) ClearDetails() {
	inst.tags = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaTagsTable()
	})
}

func (inst *LambdaTagsTable) RefreshDetails(config types.FunctionConfiguration) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.tags, err = inst.serviceCtx.Api.ListTags(aws.ToString(inst.data.FunctionArn))
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateLambdaTagsTable()
	})
}
