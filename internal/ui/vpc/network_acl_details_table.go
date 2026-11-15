package vpc

import (
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// NetworkAclDetailsTable shows the selected network ACL, which comes with the
// list, so nothing is loaded
type NetworkAclDetailsTable struct {
	*core.DetailsTable
	data *types.NetworkAcl
}

func NewNetworkAclDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *NetworkAclDetailsTable {
	var table = &NetworkAclDetailsTable{
		DetailsTable: core.NewDetailsTable("Network ACL Details", serviceCtx.AppContext),
	}

	table.populateDetailsTable()

	return table
}

func networkAclRows(acl types.NetworkAcl) []core.TableRow {
	var rows = []core.TableRow{
		{"Name", networkAclName(acl)},
		{"Network ACL ID", aws.ToString(acl.NetworkAclId)},
		{"VPC ID", aws.ToString(acl.VpcId)},
		{"Default", strconv.FormatBool(aws.ToBool(acl.IsDefault))},
		{"Owner ID", aws.ToString(acl.OwnerId)},
		{"Inbound Rules", strconv.Itoa(networkAclRuleCount(acl, false))},
		{"Outbound Rules", strconv.Itoa(networkAclRuleCount(acl, true))},
		{"Associated Subnets", strconv.Itoa(len(acl.Associations))},
	}

	var tags = core.TagRows(acl.Tags, func(t types.Tag) (string, string) {
		return aws.ToString(t.Key), aws.ToString(t.Value)
	})
	return append(rows, core.DetailsSection("Tags", tags...)...)
}

func (inst *NetworkAclDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = networkAclRows(*inst.data)
		inst.SetTitleExtra(aws.ToString(inst.data.NetworkAclId))
		inst.SetRawData(*inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetNetworkAcl shows the network ACL, or nothing for an empty one
func (inst *NetworkAclDetailsTable) SetNetworkAcl(acl types.NetworkAcl) {
	if acl.NetworkAclId == nil {
		inst.data = nil
	} else {
		inst.data = &acl
	}
	inst.populateDetailsTable()
}
