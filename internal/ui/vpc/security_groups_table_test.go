package vpc

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var securityGroupHeadings = core.TableRow{
	"Name", "Group Id", "Group Name", "Description", "Inbound Rule Count", "Outbound Rule Count", "Owner Id",
}

func newSecurityGroup(id string, tags []types.Tag) types.SecurityGroup {
	return types.SecurityGroup{
		GroupId:             aws.String(id),
		GroupName:           aws.String("web"),
		Description:         aws.String("web servers"),
		OwnerId:             aws.String("123456789012"),
		IpPermissions:       []types.IpPermission{{}, {}},
		IpPermissionsEgress: []types.IpPermission{{}},
		Tags:                tags,
	}
}

func newPopulatedSecurityGroupsTable(groups ...types.SecurityGroup) *VpcSecurityGroupsTable {
	var table = NewVpcSecurityGroupsTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.populateVpcSecurityGroupsTable(groups)
	return table
}

func TestSecurityGroupsTable__ShowsHeadingsAndRuleCounts(t *testing.T) {
	var table = newPopulatedSecurityGroupsTable(newSecurityGroup("sg-1", nameTag("web-sg")))

	uitest.AssertTableRows(t, table, []core.TableRow{
		securityGroupHeadings,
		{"web-sg", "sg-1", "web", "web servers", "2", "1", "123456789012"},
	})
}

func TestSecurityGroupsTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedSecurityGroupsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{securityGroupHeadings})
}

func TestSecurityGroupsTable__SelectingRowTracksSelectedGroup(t *testing.T) {
	var table = newPopulatedSecurityGroupsTable(
		newSecurityGroup("sg-1", nil),
		newSecurityGroup("sg-2", nil),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSeletedSecurityGroup().GroupId); got != "sg-2" {
		t.Errorf("selected group = %q, want %q", got, "sg-2")
	}
}
