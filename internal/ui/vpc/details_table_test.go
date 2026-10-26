package vpc

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func newDetailsTable() *VpcDetailsTable {
	return NewVpcDetailsTable(uitest.NewServiceContext[awsapi.Ec2Api]())
}

func TestVpcDetailsTable__NoVpcShowsNothing(t *testing.T) {
	var table = newDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, nil)
}

func TestVpcDetailsTable__ShowsTheVpcCidrBlocksAndSortedTags(t *testing.T) {
	var table = newDetailsTable()
	var vpc = newVpc("vpc-1", []types.Tag{
		{Key: aws.String("team"), Value: aws.String("orders")},
		{Key: aws.String("Name"), Value: aws.String("main")},
	})
	vpc.IsDefault = aws.Bool(true)
	vpc.CidrBlockAssociationSet = []types.VpcCidrBlockAssociation{{
		CidrBlock:      aws.String("10.0.0.0/16"),
		CidrBlockState: &types.VpcCidrBlockState{State: types.VpcCidrBlockStateCodeAssociated},
	}}

	table.SetVpc(vpc)

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "main"},
		{"VPC ID", "vpc-1"},
		{"State", "available"},
		{"Default", "true"},
		{"Owner ID", "123456789012"},
		{"DHCP Options ID", "dopt-1"},
		{"Instance Tenancy", "default"},
		{"IPv4 CIDR Blocks"},
		{"10.0.0.0/16", "associated"},
		{"Tags"},
		{"Name", "main"},
		{"team", "orders"},
	})
	if got := table.GetTitle(); got != "VPC Details ❬vpc-1❭" {
		t.Errorf("title = %q", got)
	}
}

func TestHomeView__SelectedVpcIsShownInTheDetails(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = &fakeEc2Api{vpcs: []types.Vpc{newVpc("vpc-orders", nameTag("orders"))}}
	var screen = uitest.ShowPage(t, appCtx.App, NewVpcHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "VPC Details")
	uitest.AwaitScreenText(t, appCtx.App, screen, "Instance Tenancy")
}
