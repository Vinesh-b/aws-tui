package servicetables

import (
	"sort"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBusTagsTable struct {
	*core.DetailsTable
	data       types.EventBus
	tags       []types.Tag
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBusTagsTable(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBusTagsTable {
	var table = &EventBusTagsTable{
		DetailsTable: core.NewDetailsTable("Tags", serviceCtx.AppContext),
		data:         types.EventBus{},
		serviceCtx:   serviceCtx,
	}

	table.populateEventBusTagsTable()

	return table
}

func (inst *EventBusTagsTable) populateEventBusTagsTable() {
	var tableData []core.TableRow
	for _, t := range inst.tags {
		tableData = append(tableData, core.TableRow{
			aws.ToString(t.Key), aws.ToString(t.Value),
		})
	}

	sort.Slice(tableData, func(i int, j int) bool {
		return tableData[i][0] < tableData[j][0]
	})

	inst.SetTitleExtra(aws.ToString(inst.data.Name))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *EventBusTagsTable) ClearDetails() {
	inst.tags = nil
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	dataLoader.AsyncLoadData(func() {})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventBusTagsTable()
	})
}

func (inst *EventBusTagsTable) RefreshDetails(config types.EventBus) {
	inst.data = config

	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		inst.tags, err = inst.serviceCtx.Api.ListTags(false, aws.ToString(inst.data.Arn))
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventBusTagsTable()
	})
}
