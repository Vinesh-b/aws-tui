package vpc

import (
	"errors"
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
	"aws-tui/internal/ui/vpc/vpcfake"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var subnetsPageHeadings = core.TableRow{
	"Name", "Subnet Id", "VPC Id", "State", "IPv4 CIDR Block", "Available IPv4 addresses", "AZ", "Default",
}

// newVpcSubnet is a subnet of the VPC, with an IPv6 block and a tag besides
// its name
func newVpcSubnet(id string, vpcId string, cidr string, name string) types.Subnet {
	var subnet = newSubnet(id, []types.Tag{
		{Key: aws.String("tier"), Value: aws.String("app")},
		{Key: aws.String("Name"), Value: aws.String(name)},
	})
	subnet.VpcId = aws.String(vpcId)
	subnet.CidrBlock = aws.String(cidr)
	subnet.SubnetArn = aws.String("arn:aws:ec2:us-east-1:123456789012:subnet/" + id)
	subnet.AvailabilityZoneId = aws.String("use1-az1")
	subnet.DefaultForAz = aws.Bool(false)
	subnet.MapPublicIpOnLaunch = aws.Bool(true)
	subnet.AssignIpv6AddressOnCreation = aws.Bool(false)
	subnet.Ipv6CidrBlockAssociationSet = []types.SubnetIpv6CidrBlockAssociation{{
		Ipv6CidrBlock:      aws.String("2600:1f18::/64"),
		Ipv6CidrBlockState: &types.SubnetCidrBlockState{State: types.SubnetCidrBlockStateCodeAssociated},
	}}
	return subnet
}

func newSubnetsTest(t *testing.T, api *vpcfake.Api) *SubnetsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var view = NewSubnetsPageView(NewSubnetsTable(serviceCtx), NewSubnetDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func (inst *SubnetsPageView) app() *core.AppContext {
	return inst.SubnetsTable.serviceCtx.AppContext
}

func (inst *SubnetsPageView) reset(t *testing.T) {
	t.Helper()
	uitest.OnUI(t, inst.app().App, func() { uitest.PressTableRune(inst.SubnetsTable, core.APP_KEY_BINDINGS.Reset) })
}

func TestSubnetsTable__ResetLoadsTheSubnetsOfEveryVpc(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithSubnets([]types.Subnet{
		newVpcSubnet("subnet-a", "vpc-1", "10.0.1.0/24", "private-a"),
		newVpcSubnet("subnet-b", "vpc-2", "10.1.1.0/24", "public-b"),
	}))
	var view = newSubnetsTest(t, api)

	view.reset(t)

	uitest.AwaitRows(t, view.app().App, view.SubnetsTable, []core.TableRow{
		subnetsPageHeadings,
		{"private-a", "subnet-a", "vpc-1", "available", "10.0.1.0/24", "251", "us-east-1a", "false"},
		{"public-b", "subnet-b", "vpc-2", "available", "10.1.1.0/24", "251", "us-east-1a", "false"},
	})
}

func TestSubnetsPage__SelectedSubnetIsShownInTheDetails(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithSubnets([]types.Subnet{
		newVpcSubnet("subnet-a", "vpc-1", "10.0.1.0/24", "private-a"),
	}))
	var view = newSubnetsTest(t, api)

	view.reset(t)

	uitest.AwaitDetailsRows(t, view.app().App, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "private-a"},
		{"Subnet ID", "subnet-a"},
		{"Subnet ARN", "arn:aws:ec2:us-east-1:123456789012:subnet/subnet-a"},
		{"VPC ID", "vpc-1"},
		{"State", "available"},
		{"Type", "private"},
		{"Availability Zone", "us-east-1a"},
		{"Availability Zone ID", "use1-az1"},
		{"IPv4 CIDR Block", "10.0.1.0/24"},
		{"Available IPv4 Addresses", "251"},
		{"Default For AZ", "false"},
		{"Auto-assign Public IPv4", "true"},
		{"Auto-assign IPv6", "false"},
		{"IPv6 Only", "false"},
		{"DNS64", "false"},
		{"Block Public Access", "block-bidirectional"},
		{"Owner ID", "123456789012"},
		{"IPv6 CIDR Blocks"},
		{"2600:1f18::/64", "associated"},
		{"Tags"},
		{"Name", "private-a"},
		{"tier", "app"},
	})
	uitest.OnUI(t, view.app().App, func() {
		if got := view.DetailsTable.GetTitle(); got != "Subnet Details ❬subnet-a❭" {
			t.Errorf("title = %q", got)
		}
	})
}

func TestSubnetsPage__MovingTheSelectionShowsTheNextSubnet(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithSubnets([]types.Subnet{
		newVpcSubnet("subnet-a", "vpc-1", "10.0.1.0/24", "private-a"),
		newVpcSubnet("subnet-b", "vpc-2", "10.1.1.0/24", "public-b"),
	}))
	var view = newSubnetsTest(t, api)
	view.reset(t)
	uitest.AwaitDetailsTag(t, view.app().App, view.DetailsTable.DetailsTable, "Name", "private-a")

	uitest.OnUI(t, view.app().App, func() { view.SubnetsTable.Select(2, 0) })

	uitest.Eventually(t, view.app().App, func() bool {
		var row = uitest.DetailsRowOf(view.DetailsTable.DetailsTable, "Subnet ID")
		return row >= 0 && view.DetailsTable.GetCellText(row, 1) == "subnet-b"
	}, "the details are of subnet-b")
}

func TestSubnetsPage__RefreshWithNoSubnetsClearsTheDetails(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithSubnets([]types.Subnet{
		newVpcSubnet("subnet-a", "vpc-1", "10.0.1.0/24", "private-a"),
	}))
	var view = newSubnetsTest(t, api)
	view.reset(t)
	uitest.AwaitDetailsTag(t, view.app().App, view.DetailsTable.DetailsTable, "Name", "private-a")

	api.SetSubnets(nil)
	view.reset(t)

	uitest.AwaitRows(t, view.app().App, view.SubnetsTable, []core.TableRow{subnetsPageHeadings})
	uitest.Eventually(t, view.app().App, func() bool {
		return uitest.DetailsRowOf(view.DetailsTable.DetailsTable, "Subnet ID") < 0
	}, "the details are cleared")
}

func TestSubnetsTable__RefreshReportsTheApiError(t *testing.T) {
	var api = vpcfake.New(vpcfake.WithErr(errors.New("not authorized to describe subnets")))
	var view = newSubnetsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.SubnetsTable.ErrorMessageCallback = callback

	view.reset(t)

	if got := uitest.AwaitError(t, received); got != "not authorized to describe subnets" {
		t.Errorf("message = %q", got)
	}
}

func TestHomeView__SubnetsPageLoadsTheSubnets(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = vpcfake.New(vpcfake.WithSubnets([]types.Subnet{
		newVpcSubnet("subnet-a", "vpc-1", "10.0.1.0/24", "private-orders-a"),
	}))
	var page = NewVpcHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	var paged, ok = page.(core.PagedServicePage)
	if !ok {
		t.Fatal("the VPC home view has no pages")
	}
	if pages := paged.ServicePages(); !slices.Contains(pages, "Subnets") {
		t.Fatalf("pages = %v, want a Subnets page", pages)
	}
	uitest.OnUI(t, appCtx.App, func() { paged.ShowServicePage("Subnets") })
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "private-orders-a")
	if got := api.SubnetListings(); got != 1 {
		t.Errorf("listings = %d, want 1", got)
	}
}
