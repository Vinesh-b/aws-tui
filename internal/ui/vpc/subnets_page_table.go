package vpc

import (
	"context"
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gdamore/tcell/v2"
)

// SubnetsTable lists the subnets of every VPC in the region. Each comes with
// all its details, so nothing more is loaded for the one selected.
type SubnetsTable struct {
	*core.SelectableTable[types.Subnet]
	data           []types.Subnet
	selectedSubnet types.Subnet
	serviceCtx     *core.ServiceContext[awsapi.Ec2Api]
}

func NewSubnetsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *SubnetsTable {
	var table = &SubnetsTable{
		SelectableTable: core.NewSelectableTable[types.Subnet](
			"Subnets",
			core.TableRow{
				"Name",
				"Subnet Id",
				"VPC Id",
				"State",
				"IPv4 CIDR Block",
				"Available IPv4 addresses",
				"AZ",
				"Default",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1, 2),
		serviceCtx: serviceCtx,
	}

	table.populateSubnetsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func subnetName(subnet types.Subnet) string {
	for _, tag := range subnet.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func (inst *SubnetsTable) populateSubnetsTable() {
	var tableData []core.TableRow
	for _, subnet := range inst.data {
		tableData = append(tableData, core.TableRow{
			subnetName(subnet),
			aws.ToString(subnet.SubnetId),
			aws.ToString(subnet.VpcId),
			string(subnet.State),
			aws.ToString(subnet.CidrBlock),
			strconv.Itoa(int(aws.ToInt32(subnet.AvailableIpAddressCount))),
			aws.ToString(subnet.AvailabilityZone),
			strconv.FormatBool(aws.ToBool(subnet.DefaultForAz)),
		})
	}

	// Selecting the first row of an empty table selects an empty subnet,
	// which clears the details of the one that was selected
	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshSubnets loads the subnets
func (inst *SubnetsTable) RefreshSubnets() {
	var subnets []types.Subnet
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		subnets, err = inst.serviceCtx.Api.ListSubnets(ctx, true)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = subnets
		inst.populateSubnetsTable()
	})
}

func (inst *SubnetsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSubnet = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *SubnetsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshSubnets()
			return nil
		}
		return capture(event)
	})
}

func (inst *SubnetsTable) GetSelectedSubnet() types.Subnet {
	return inst.selectedSubnet
}
