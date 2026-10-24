package vpc

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var subnetHeadings = core.TableRow{
	"Name", "Subnet Id", "Status", "IPv4 CIDR Block", "Available IPv4 addresses",
	"Block Public Access", "AZ", "Owner Id", "Type",
}

func newSubnet(id string, tags []types.Tag) types.Subnet {
	return types.Subnet{
		SubnetId:                aws.String(id),
		State:                   types.SubnetStateAvailable,
		CidrBlock:               aws.String("10.0.1.0/24"),
		AvailableIpAddressCount: aws.Int32(251),
		AvailabilityZone:        aws.String("us-east-1a"),
		OwnerId:                 aws.String("123456789012"),
		Type:                    aws.String("private"),
		Tags:                    tags,
		BlockPublicAccessStates: &types.BlockPublicAccessStates{
			InternetGatewayBlockMode: types.BlockPublicAccessModeBlockBidirectional,
		},
	}
}

func newPopulatedSubnetsTable(subnets ...types.Subnet) *VpcSubnetsTable {
	var table = NewVpcSubnetsTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.populateVpcSubnetsTable(subnets)
	return table
}

func TestSubnetsTable__ShowsHeadingsAndOneRowPerSubnet(t *testing.T) {
	var table = newPopulatedSubnetsTable(newSubnet("subnet-1", nameTag("private-a")))

	uitest.AssertTableRows(t, table, []core.TableRow{
		subnetHeadings,
		{
			"private-a", "subnet-1", "available", "10.0.1.0/24", "251",
			"block-bidirectional", "us-east-1a", "123456789012", "private",
		},
	})
}

func TestSubnetsTable__MissingBlockPublicAccessStatesShowsBlank(t *testing.T) {
	var subnet = newSubnet("subnet-1", nil)
	subnet.BlockPublicAccessStates = nil

	var table = newPopulatedSubnetsTable(subnet)

	if got := table.GetCellText(1, 5); got != "" {
		t.Errorf("block public access = %q, want empty", got)
	}
}

func TestSubnetsTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedSubnetsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{subnetHeadings})
}

func TestSubnetsTable__SelectingRowTracksSelectedSubnet(t *testing.T) {
	var table = newPopulatedSubnetsTable(
		newSubnet("subnet-1", nil),
		newSubnet("subnet-2", nil),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSeletedVpcEndpoint().SubnetId); got != "subnet-2" {
		t.Errorf("selected subnet = %q, want %q", got, "subnet-2")
	}
}
