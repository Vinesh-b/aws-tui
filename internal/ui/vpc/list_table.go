package vpc

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gdamore/tcell/v2"
)

type VpcListTable struct {
	*core.SelectableTable[types.Vpc]
	data        []types.Vpc
	selectedVpc types.Vpc
	serviceCtx  *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcListTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcListTable {

	var view = &VpcListTable{
		SelectableTable: core.NewSelectableTable[types.Vpc](
			"VPCs",
			core.TableRow{
				"Name",
				"Vpc Id",
				"State",
				"CIDR Block",
				"Owner Id",
				"DHCP Options Id",
				"Instance Tenancy",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		data:        nil,
		selectedVpc: types.Vpc{},
		serviceCtx:  serviceCtx,
	}

	view.populateVpcTable(view.data)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectedFunc(func(row, column int) {})
	view.SetSelectionChangedFunc(func(row, column int) {})

	return view
}

func (inst *VpcListTable) populateVpcTable(data []types.Vpc) {
	var tableData []core.TableRow
	var privateData []types.Vpc

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
			aws.ToString(row.VpcId),
			string(row.State),
			aws.ToString(row.CidrBlock),
			aws.ToString(row.OwnerId),
			aws.ToString(row.DhcpOptionsId),
			string(row.InstanceTenancy),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetTable().SetFixed(1, 0)
	inst.Select(1, 0)
}

func (inst *VpcListTable) RefreshVpcs(reset bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.ListVpcs(inst.serviceCtx.Ctx, reset)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}

		if !reset {
			inst.data = append(inst.data, data...)
		} else {
			inst.data = data
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcTable(inst.data)
	})
}

func (inst *VpcListTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpc = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcListTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedVpc = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcListTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshVpcs(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *VpcListTable) GetSelectedVpc() types.Vpc {
	return inst.selectedVpc
}
