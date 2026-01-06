package servicetables

import (
	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"
	"aws-tui/internal/pkg/utils"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gdamore/tcell/v2"
)

type VpcSubnetsTable struct {
	*core.SelectableTable[types.Subnet]
	data              []types.Subnet
	filtered          []types.Subnet
	selectedVpcSubnet types.Subnet
	selectedVpc       types.Vpc
	serviceCtx        *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcSubnetsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcSubnetsTable {

	var view = &VpcSubnetsTable{
		SelectableTable: core.NewSelectableTable[types.Subnet](
			"VPC Subnets",
			core.TableRow{
				"Name",
				"Subnet Id",
				"Status",
				"IPv4 CIDR Block",
				"Available IPv4 addresses",
				"Block Public Access",
				"AZ",
				"Owner Id",
				"Type",
			},
			serviceCtx.AppContext,
		),
		data:              nil,
		selectedVpcSubnet: types.Subnet{},
		serviceCtx:        serviceCtx,
	}

	view.populateVpcSubnetsTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *VpcSubnetsTable) populateVpcSubnetsTable(data []types.Subnet) {
	var tableData []core.TableRow
	var privateData []types.Subnet
	for _, row := range data {
		var name = ""
		for _, t := range row.Tags {
			if aws.ToString(t.Key) == "Name" {
				name = aws.ToString(t.Value)
				break
			}
		}
		tableData = append(tableData, core.TableRow{
			name,
			aws.ToString(row.SubnetId),
			string(row.State),
			aws.ToString(row.CidrBlock),
			fmt.Sprintf("%d", aws.ToInt32(row.AvailableIpAddressCount)),
			string(row.BlockPublicAccessStates.InternetGatewayBlockMode),
			aws.ToString(row.AvailabilityZone),
			aws.ToString(row.OwnerId),
			aws.ToString(row.Type),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.GetTable().SetFixed(1, 0)
}

func (inst *VpcSubnetsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(ep types.Subnet) string {
				for _, t := range ep.Tags {
					if aws.ToString(t.Key) == "Name" {
						return aws.ToString(t.Value)
					}
				}

				return aws.ToString(ep.SubnetId)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcSubnetsTable(inst.filtered)
	})
}

func (inst *VpcSubnetsTable) RefreshVpcSubnets(reset bool, vpc types.Vpc) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	inst.selectedVpc = vpc

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.DescribeVpcSubnets(
			reset, aws.ToString(inst.selectedVpc.VpcId),
		)
		if err != nil {
			inst.ErrorMessageCallback(err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcSubnetsTable(inst.data)
	})
}

func (inst *VpcSubnetsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpcSubnet = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcSubnetsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpcSubnet = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcSubnetsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshVpcSubnets(true, inst.selectedVpc)
			return nil
		}
		return capture(event)
	})
}

func (inst *VpcSubnetsTable) GetSeletedVpcEndpoint() types.Subnet {
	return inst.selectedVpcSubnet
}
