package vpc

import (
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// VpcDetailsTable shows the selected VPC. Everything it shows, tags included,
// comes with the VPC in the list, so nothing is loaded.
type VpcDetailsTable struct {
	*core.DetailsTable
	data *types.Vpc
}

func NewVpcDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *VpcDetailsTable {
	var table = &VpcDetailsTable{
		DetailsTable: core.NewDetailsTable("VPC Details", serviceCtx.AppContext),
	}

	table.populateDetailsTable()

	return table
}

func vpcName(vpc types.Vpc) string {
	for _, tag := range vpc.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func vpcRows(vpc types.Vpc) []core.TableRow {
	var rows = []core.TableRow{
		{"Name", vpcName(vpc)},
		{"VPC ID", aws.ToString(vpc.VpcId)},
		{"State", string(vpc.State)},
		{"Default", strconv.FormatBool(aws.ToBool(vpc.IsDefault))},
		{"Owner ID", aws.ToString(vpc.OwnerId)},
		{"DHCP Options ID", aws.ToString(vpc.DhcpOptionsId)},
		{"Instance Tenancy", string(vpc.InstanceTenancy)},
	}

	var cidrs []core.TableRow
	for _, association := range vpc.CidrBlockAssociationSet {
		var state string
		if association.CidrBlockState != nil {
			state = string(association.CidrBlockState.State)
		}
		cidrs = append(cidrs, core.TableRow{aws.ToString(association.CidrBlock), state})
	}
	rows = append(rows, core.DetailsSection("IPv4 CIDR Blocks", cidrs...)...)

	var ipv6Cidrs []core.TableRow
	for _, association := range vpc.Ipv6CidrBlockAssociationSet {
		var state string
		if association.Ipv6CidrBlockState != nil {
			state = string(association.Ipv6CidrBlockState.State)
		}
		ipv6Cidrs = append(ipv6Cidrs, core.TableRow{aws.ToString(association.Ipv6CidrBlock), state})
	}
	rows = append(rows, core.DetailsSection("IPv6 CIDR Blocks", ipv6Cidrs...)...)

	var tags = core.TagRows(vpc.Tags, func(t types.Tag) (string, string) {
		return aws.ToString(t.Key), aws.ToString(t.Value)
	})
	return append(rows, core.DetailsSection("Tags", tags...)...)
}

func (inst *VpcDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = vpcRows(*inst.data)
		inst.SetTitleExtra(aws.ToString(inst.data.VpcId))
		inst.SetRawData(*inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetVpc shows the VPC
func (inst *VpcDetailsTable) SetVpc(vpc types.Vpc) {
	inst.data = &vpc
	inst.populateDetailsTable()
}
