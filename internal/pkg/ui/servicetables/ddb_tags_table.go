package servicetables

import (
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBTagsTable struct {
	*core.DetailsTable
	data       types.TableDescription
	tags       []types.Tag
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi]
}

func NewDynamoDbTagsTable(
	serviceCtx *core.ServiceContext[awsapi.DynamoDBApi],
) *DynamoDBTagsTable {
	var table = &DynamoDBTagsTable{
		DetailsTable: core.NewDetailsTable("Tags", serviceCtx.AppContext),
		data:         types.TableDescription{},
		serviceCtx:   serviceCtx,
	}

	table.populateDynamoDBTagsTable()

	return table
}

func (inst *DynamoDBTagsTable) populateDynamoDBTagsTable() {
	var tableData []core.TableRow
	for _, t := range inst.tags {
		tableData = append(tableData, core.TableRow{
			aws.ToString(t.Key), aws.ToString(t.Value),
		})
	}

	sort.Slice(tableData, func(i int, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetTitleExtra(aws.ToString(inst.data.TableName))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *DynamoDBTagsTable) ClearDetails() {
	inst.tags = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDynamoDBTagsTable()
	})
}

func (inst *DynamoDBTagsTable) RefreshDetails(config types.TableDescription) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.tags, err = inst.serviceCtx.Api.ListTags(false, aws.ToString(inst.data.TableArn))
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateDynamoDBTagsTable()
	})
}
