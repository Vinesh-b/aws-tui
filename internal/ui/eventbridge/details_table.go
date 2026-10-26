package eventbridge

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBusDetailsTable struct {
	*core.DetailsTable
	data       types.EventBus
	tags       core.DetailsTags
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi]
}

func NewEventBusDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.EventBridgeApi],
) *EventBusDetailsTable {
	var table = &EventBusDetailsTable{
		DetailsTable: core.NewDetailsTable("EventBus Details", serviceCtx.AppContext),
		data:         types.EventBus{},
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
		{"ARN", aws.ToString(inst.data.Arn)},
		{"Last Modified", core.FormatTime(inst.data.LastModifiedTime)},
	}

	tableData = append(tableData, inst.tags.Section(aws.ToString(inst.data.Arn))...)

	inst.SetTitleExtra(aws.ToString(inst.data.Name))
	inst.SetRawData(inst.data)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// RefreshDetails shows the bus, and then its tags once they are loaded
func (inst *EventBusDetailsTable) RefreshDetails(busDetail types.EventBus) {
	inst.data = busDetail
	inst.populateEventBusDetailsTable()

	var busArn = aws.ToString(busDetail.Arn)
	inst.tags.Load(inst.serviceCtx.App, inst.Box, busArn,
		func() string { return aws.ToString(inst.data.Arn) },
		func() ([]core.TableRow, error) {
			var tags, err = inst.serviceCtx.Api.ListTags(inst.serviceCtx.Ctx, true, busArn)
			return core.TagRows(tags, func(t types.Tag) (string, string) {
				return aws.ToString(t.Key), aws.ToString(t.Value)
			}), err
		},
		inst.ErrorMessageCallback,
		inst.populateEventBusDetailsTable,
	)
}
