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

type VpcSecurityGroupsTable struct {
	*core.SelectableTable[types.SecurityGroup]
	data                  []types.SecurityGroup
	filtered              []types.SecurityGroup
	selectedSecurityGroup types.SecurityGroup
	selectedVpc           types.Vpc
	serviceCtx            *core.ServiceContext[awsapi.Ec2Api]
}

func NewVpcSecurityGroupsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcSecurityGroupsTable {
	var view = &VpcSecurityGroupsTable{
		SelectableTable: core.NewSelectableTable[types.SecurityGroup](
			"VPC Security Groups",
			core.TableRow{
				"Name",
				"Group Id",
				"Group Name",
				"Description",
				"Inbound Rule Count",
				"Outbound Rule Count",
				"Owner Id",
			},
			serviceCtx.AppContext,
		),
		data:                  nil,
		selectedSecurityGroup: types.SecurityGroup{},
		serviceCtx:            serviceCtx,
	}

	view.populateVpcSecurityGroupsTable(view.data)
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

func (inst *VpcSecurityGroupsTable) populateVpcSecurityGroupsTable(data []types.SecurityGroup) {
	var tableData []core.TableRow
	var privateData []types.SecurityGroup
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
			aws.ToString(row.GroupId),
			aws.ToString(row.GroupName),
			aws.ToString(row.Description),
			fmt.Sprintf("%d", len(row.IpPermissions)),
			fmt.Sprintf("%d", len(row.IpPermissionsEgress)),
			aws.ToString(row.OwnerId),
		})
		privateData = append(privateData, row)
	}

	inst.SetData(tableData, privateData, 0)
	inst.GetCell(0, 0).SetExpansion(1)
	inst.GetTable().SetFixed(1, 0)
}

func (inst *VpcSecurityGroupsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = utils.FuzzySearch(
			name,
			inst.data,
			func(ep types.SecurityGroup) string {
				for _, t := range ep.Tags {
					if aws.ToString(t.Key) == "Name" {
						return aws.ToString(t.Value)
					}
				}

				return aws.ToString(ep.GroupId)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateVpcSecurityGroupsTable(inst.filtered)
	})
}

func (inst *VpcSecurityGroupsTable) RefreshVpcSecurityGroups(reset bool, vpc types.Vpc) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	inst.selectedVpc = vpc

	dataLoader.AsyncLoadData(func() {
		var data, err = inst.serviceCtx.Api.DescribeVpcSecurityGroups(
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
		inst.populateVpcSecurityGroupsTable(inst.data)
	})
}

func (inst *VpcSecurityGroupsTable) SetSelectedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSecurityGroup = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcSecurityGroupsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedSecurityGroup = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *VpcSecurityGroupsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset, core.APP_KEY_BINDINGS.LoadMoreData:
			inst.RefreshVpcSecurityGroups(true, inst.selectedVpc)
			return nil
		}
		return capture(event)
	})
}

func (inst *VpcSecurityGroupsTable) GetSeletedSecurityGroup() types.SecurityGroup {
	return inst.selectedSecurityGroup
}
