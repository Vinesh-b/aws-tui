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

// NetworkAclsTable lists the network ACLs of every VPC in the region. Each
// comes with its rules and subnet associations, so nothing more is loaded for
// the one selected.
type NetworkAclsTable struct {
	*core.SelectableTable[types.NetworkAcl]
	data               []types.NetworkAcl
	selectedNetworkAcl types.NetworkAcl
	serviceCtx         *core.ServiceContext[awsapi.Ec2Api]
}

func NewNetworkAclsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *NetworkAclsTable {
	var table = &NetworkAclsTable{
		SelectableTable: core.NewSelectableTable[types.NetworkAcl](
			"Network ACLs",
			core.TableRow{
				"Name",
				"Network ACL Id",
				"VPC Id",
				"Default",
				"Inbound Rules",
				"Outbound Rules",
				"Subnets",
				"Owner Id",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1, 2),
		serviceCtx: serviceCtx,
	}

	table.populateNetworkAclsTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })

	return table
}

func networkAclName(acl types.NetworkAcl) string {
	for _, tag := range acl.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// networkAclRuleCount is how many rules the ACL has in one direction
func networkAclRuleCount(acl types.NetworkAcl, egress bool) int {
	var count = 0
	for _, entry := range acl.Entries {
		if aws.ToBool(entry.Egress) == egress {
			count++
		}
	}
	return count
}

func (inst *NetworkAclsTable) populateNetworkAclsTable() {
	var tableData []core.TableRow
	for _, acl := range inst.data {
		tableData = append(tableData, core.TableRow{
			networkAclName(acl),
			aws.ToString(acl.NetworkAclId),
			aws.ToString(acl.VpcId),
			strconv.FormatBool(aws.ToBool(acl.IsDefault)),
			strconv.Itoa(networkAclRuleCount(acl, false)),
			strconv.Itoa(networkAclRuleCount(acl, true)),
			strconv.Itoa(len(acl.Associations)),
			aws.ToString(acl.OwnerId),
		})
	}

	// Selecting the first row of an empty table selects an empty ACL, which
	// clears the views of the one that was selected
	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshNetworkAcls loads the network ACLs
func (inst *NetworkAclsTable) RefreshNetworkAcls() {
	var acls []types.NetworkAcl
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.AppContext, 10)

	dataLoader.AsyncLoadData(func(ctx context.Context) {
		var err error
		acls, err = inst.serviceCtx.Api.ListNetworkAcls(ctx, true)
		if err != nil {
			inst.ErrorMessageCallback("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = acls
		inst.populateNetworkAclsTable()
	})
}

func (inst *NetworkAclsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedNetworkAcl = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *NetworkAclsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshNetworkAcls()
			return nil
		}
		return capture(event)
	})
}

func (inst *NetworkAclsTable) GetSelectedNetworkAcl() types.NetworkAcl {
	return inst.selectedNetworkAcl
}
