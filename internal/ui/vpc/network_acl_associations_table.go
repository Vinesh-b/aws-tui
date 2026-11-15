package vpc

import (
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// NetworkAclAssociationsTable shows the subnets the selected network ACL is
// associated with
type NetworkAclAssociationsTable struct {
	*core.SelectableTable[types.NetworkAclAssociation]
}

func NewNetworkAclAssociationsTable(serviceCtx *core.ServiceContext[awsapi.Ec2Api]) *NetworkAclAssociationsTable {
	var table = &NetworkAclAssociationsTable{
		SelectableTable: core.NewSelectableTable[types.NetworkAclAssociation](
			"Subnet Associations",
			core.TableRow{
				"Subnet Id",
				"Association Id",
			},
			serviceCtx.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
	}
	table.SetNetworkAcl(types.NetworkAcl{})
	return table
}

// SetNetworkAcl shows the subnet associations of the network ACL
func (inst *NetworkAclAssociationsTable) SetNetworkAcl(acl types.NetworkAcl) {
	var associations = slices.Clone(acl.Associations)
	slices.SortStableFunc(associations, func(a, b types.NetworkAclAssociation) int {
		return strings.Compare(aws.ToString(a.SubnetId), aws.ToString(b.SubnetId))
	})

	var tableData []core.TableRow
	for _, association := range associations {
		tableData = append(tableData, core.TableRow{
			aws.ToString(association.SubnetId),
			aws.ToString(association.NetworkAclAssociationId),
		})
	}

	inst.SetTitleExtra(aws.ToString(acl.NetworkAclId))
	inst.SetData(tableData, associations, 0)
	inst.Select(1, 0)
	inst.ScrollToBeginning()
}
