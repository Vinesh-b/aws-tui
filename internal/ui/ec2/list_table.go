package ec2

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gdamore/tcell/v2"
)

func instanceName(instance types.Instance) string {
	for _, tag := range instance.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func instanceState(instance types.Instance) string {
	if instance.State == nil {
		return ""
	}
	return string(instance.State.Name)
}

type Ec2InstancesTable struct {
	*core.SelectableTable[types.Instance]
	data             []types.Instance
	selectedInstance types.Instance
	serviceCtx       *core.ServiceContext[awsapi.Ec2Api]
}

func NewEc2InstancesTable(
	serviceContext *core.ServiceContext[awsapi.Ec2Api],
) *Ec2InstancesTable {

	var table = &Ec2InstancesTable{
		SelectableTable: core.NewSelectableTable[types.Instance](
			"EC2 Instances",
			core.TableRow{
				"Name",
				"Instance ID",
				"State",
				"Type",
				"Availability Zone",
				"Private IP",
				"Public IP",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		serviceCtx: serviceContext,
	}

	table.populateInstancesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})

	return table
}

func (inst *Ec2InstancesTable) populateInstancesTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		var zone string
		if row.Placement != nil {
			zone = aws.ToString(row.Placement.AvailabilityZone)
		}
		tableData = append(tableData, core.TableRow{
			instanceName(row),
			aws.ToString(row.InstanceId),
			instanceState(row),
			string(row.InstanceType),
			zone,
			aws.ToString(row.PrivateIpAddress),
			aws.ToString(row.PublicIpAddress),
		})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedInstance = types.Instance{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.GetTable().SetFixed(1, 0)
	inst.Select(1, 0)
}

// RefreshInstances loads the instances
func (inst *Ec2InstancesTable) RefreshInstances(force bool) {
	var data []types.Instance
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var err error
		data, err = inst.serviceCtx.Api.ListInstances(inst.serviceCtx.Ctx, force)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = data
		inst.populateInstancesTable()
	})
}

func (inst *Ec2InstancesTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedInstance = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *Ec2InstancesTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshInstances(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *Ec2InstancesTable) GetSelectedInstance() types.Instance {
	return inst.selectedInstance
}
