package servicetables

import (
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type EventBusDetailsTable struct {
	*core.DetailsTable
	data       eventbridge.DescribeEventBusOutput
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBusDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBusDetailsTable {
	var table = &EventBusDetailsTable{
		DetailsTable: core.NewDetailsTable("EventBus Details", serviceCtx.AppContext),
		data:         eventbridge.DescribeEventBusOutput{},
		serviceCtx:   serviceCtx,
	}

	table.populateEventBusDetailsTable()

	return table
}

func (inst *EventBusDetailsTable) populateEventBusDetailsTable() {
	var tableData []core.TableRow

	tableData = []core.TableRow{
		{"Name", aws.ToString(inst.data.Name)},
		{"Description", aws.ToString(inst.data.Description)},
		{"Arn", aws.ToString(inst.data.Arn)},
		{"LastModified", aws.ToTime(inst.data.LastModifiedTime).Format(time.DateTime)},
	}

	inst.SetTitleExtra(aws.ToString(inst.data.Name))
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

func (inst *EventBusDetailsTable) RefreshDetails(busArn string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.DescribeEventBus(true, busArn)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}
		inst.data = data

	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateEventBusDetailsTable()
	})
}

func (inst *EventBusDetailsTable) GetPolicy() string {
	return aws.ToString(inst.data.Policy)
}
