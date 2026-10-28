package ec2

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

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
	allInstances     []types.Instance
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
		),
		serviceCtx: serviceContext,
	}

	table.populateInstancesTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			table.RefreshInstances(false)
		}
	})

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
	inst.GetCell(0, 0).SetExpansion(1)
	inst.GetTable().SetFixed(1, 0)
	inst.Select(1, 0)
}

// RefreshInstances loads the instances, or only searches the instances that are
// here when there is search text and force is not set.
func (inst *Ec2InstancesTable) RefreshInstances(force bool) {
	var search = inst.GetSearchText()
	var allInstances = inst.allInstances
	var data []types.Instance
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	// The instances are only handled on the UI goroutine, so a refresh that is
	// still loading can't change what another one searches
	dataLoader.AsyncLoadData(func() {
		if len(search) == 0 || force || allInstances == nil {
			var err error
			allInstances, err = inst.serviceCtx.Api.ListInstances(inst.serviceCtx.Ctx, force)
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
			}
		}

		if len(search) > 0 {
			data = utils.FuzzySearch(search, allInstances, func(i types.Instance) string {
				return instanceName(i) + " " + aws.ToString(i.InstanceId)
			})
		} else {
			data = allInstances
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.allInstances = allInstances
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
