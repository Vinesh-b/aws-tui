package vpc

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var vpcHeadings = core.TableRow{
	"Name", "Vpc Id", "State", "CIDR Block", "Owner Id", "DHCP Options Id", "Instance Tenancy",
}

func newVpc(id string, tags []types.Tag) types.Vpc {
	return types.Vpc{
		VpcId:           aws.String(id),
		State:           types.VpcStateAvailable,
		CidrBlock:       aws.String("10.0.0.0/16"),
		OwnerId:         aws.String("123456789012"),
		DhcpOptionsId:   aws.String("dopt-1"),
		InstanceTenancy: types.TenancyDefault,
		Tags:            tags,
	}
}

func newPopulatedVpcListTable(vpcs ...types.Vpc) *VpcListTable {
	var table = NewVpcListTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.populateVpcTable(vpcs)
	return table
}

func TestVpcListTable__ShowsHeadingsAndOneRowPerVpc(t *testing.T) {
	var table = newPopulatedVpcListTable(newVpc("vpc-1", nameTag("main")))

	uitest.AssertTableRows(t, table, []core.TableRow{
		vpcHeadings,
		{"main", "vpc-1", "available", "10.0.0.0/16", "123456789012", "dopt-1", "default"},
	})
}

func TestVpcListTable__NameIsBlankWithoutNameTag(t *testing.T) {
	var otherTag = []types.Tag{{Key: aws.String("Env"), Value: aws.String("prod")}}
	var table = newPopulatedVpcListTable(newVpc("vpc-1", otherTag))

	if got := table.GetCellText(1, 0); got != "" {
		t.Errorf("name = %q, want empty", got)
	}
}

func TestVpcListTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedVpcListTable()

	uitest.AssertTableRows(t, table, []core.TableRow{vpcHeadings})
}

func TestVpcListTable__SelectingRowTracksSelectedVpc(t *testing.T) {
	var table = newPopulatedVpcListTable(
		newVpc("vpc-1", nameTag("main")),
		newVpc("vpc-2", nameTag("staging")),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSeletedVpc().VpcId); got != "vpc-2" {
		t.Errorf("selected vpc = %q, want %q", got, "vpc-2")
	}
}
