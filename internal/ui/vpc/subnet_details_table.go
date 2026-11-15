package vpc

import (
	"strconv"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SubnetDetailsTable shows the selected subnet, which comes with the list, so
// nothing is loaded
type SubnetDetailsTable struct {
	*core.DetailsTable
	data *types.Subnet
}

func NewSubnetDetailsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *SubnetDetailsTable {
	var table = &SubnetDetailsTable{
		DetailsTable: core.NewDetailsTable("Subnet Details", serviceCtx.AppContext),
	}

	table.populateDetailsTable()

	return table
}

func subnetRows(subnet types.Subnet) []core.TableRow {
	var blockMode string
	if subnet.BlockPublicAccessStates != nil {
		blockMode = string(subnet.BlockPublicAccessStates.InternetGatewayBlockMode)
	}

	var rows = []core.TableRow{
		{"Name", subnetName(subnet)},
		{"Subnet ID", aws.ToString(subnet.SubnetId)},
		{"Subnet ARN", aws.ToString(subnet.SubnetArn)},
		{"VPC ID", aws.ToString(subnet.VpcId)},
		{"State", string(subnet.State)},
		{"Type", aws.ToString(subnet.Type)},
		{"Availability Zone", aws.ToString(subnet.AvailabilityZone)},
		{"Availability Zone ID", aws.ToString(subnet.AvailabilityZoneId)},
		{"IPv4 CIDR Block", aws.ToString(subnet.CidrBlock)},
		{"Available IPv4 Addresses", strconv.Itoa(int(aws.ToInt32(subnet.AvailableIpAddressCount)))},
		{"Default For AZ", strconv.FormatBool(aws.ToBool(subnet.DefaultForAz))},
		{"Auto-assign Public IPv4", strconv.FormatBool(aws.ToBool(subnet.MapPublicIpOnLaunch))},
		{"Auto-assign IPv6", strconv.FormatBool(aws.ToBool(subnet.AssignIpv6AddressOnCreation))},
		{"IPv6 Only", strconv.FormatBool(aws.ToBool(subnet.Ipv6Native))},
		{"DNS64", strconv.FormatBool(aws.ToBool(subnet.EnableDns64))},
		{"Block Public Access", blockMode},
		{"Owner ID", aws.ToString(subnet.OwnerId)},
	}

	var ipv6Cidrs []core.TableRow
	for _, association := range subnet.Ipv6CidrBlockAssociationSet {
		var state string
		if association.Ipv6CidrBlockState != nil {
			state = string(association.Ipv6CidrBlockState.State)
		}
		ipv6Cidrs = append(ipv6Cidrs, core.TableRow{aws.ToString(association.Ipv6CidrBlock), state})
	}
	rows = append(rows, core.DetailsSection("IPv6 CIDR Blocks", ipv6Cidrs...)...)

	var tags = core.TagRows(subnet.Tags, func(t types.Tag) (string, string) {
		return aws.ToString(t.Key), aws.ToString(t.Value)
	})
	return append(rows, core.DetailsSection("Tags", tags...)...)
}

func (inst *SubnetDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = subnetRows(*inst.data)
		inst.SetTitleExtra(aws.ToString(inst.data.SubnetId))
		inst.SetRawData(*inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetSubnet shows the subnet, or nothing for an empty one
func (inst *SubnetDetailsTable) SetSubnet(subnet types.Subnet) {
	if subnet.SubnetId == nil {
		inst.data = nil
	} else {
		inst.data = &subnet
	}
	inst.populateDetailsTable()
}
