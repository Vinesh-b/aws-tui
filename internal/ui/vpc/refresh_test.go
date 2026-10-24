package vpc

import (
	"context"
	"errors"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type fakeEc2Api struct {
	awsapi.Ec2Api
	vpcs      []types.Vpc
	endpoints []types.VpcEndpoint
	subnets   []types.Subnet
	groups    []types.SecurityGroup
	err       error
	vpcIds    []string
}

func (f *fakeEc2Api) ListVpcs(ctx context.Context, force bool) ([]types.Vpc, error) {
	return f.vpcs, f.err
}

func (f *fakeEc2Api) DescribeVpcEndpoints(ctx context.Context, force bool, vpcId string) ([]types.VpcEndpoint, error) {
	f.vpcIds = append(f.vpcIds, vpcId)
	return f.endpoints, f.err
}

func (f *fakeEc2Api) DescribeVpcSubnets(ctx context.Context, force bool, vpcId string) ([]types.Subnet, error) {
	f.vpcIds = append(f.vpcIds, vpcId)
	return f.subnets, f.err
}

func (f *fakeEc2Api) DescribeVpcSecurityGroups(
	ctx context.Context, force bool, vpcId string,
) ([]types.SecurityGroup, error) {
	f.vpcIds = append(f.vpcIds, vpcId)
	return f.groups, f.err
}

func TestVpcListTable__RefreshLoadsVpcsFromApi(t *testing.T) {
	var api = &fakeEc2Api{vpcs: []types.Vpc{newVpc("vpc-1", nameTag("main"))}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshVpcs(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		vpcHeadings,
		{"main", "vpc-1", "available", "10.0.0.0/16", "123456789012", "dopt-1", "default"},
	})
}

func TestVpcListTable__RefreshReportsApiError(t *testing.T) {
	var api = &fakeEc2Api{err: errors.New("access denied")}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshVpcs(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestVpcListTable__FilterByNameMatchesNameTagOrFallsBackToId(t *testing.T) {
	var api = &fakeEc2Api{vpcs: []types.Vpc{newVpc("vpc-1", nameTag("main")), newVpc("vpc-2", nil)}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcListTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshVpcs(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })

	table.FilterByName("vpc-2")

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		vpcHeadings,
		{"", "vpc-2", "available", "10.0.0.0/16", "123456789012", "dopt-1", "default"},
	})
}

func TestEndpointsTable__RefreshQueriesTheGivenVpc(t *testing.T) {
	var api = &fakeEc2Api{endpoints: []types.VpcEndpoint{newEndpoint("vpce-1", nameTag("s3-gateway"))}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcEndpointsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshVpcEndpoints(true, newVpc("vpc-9", nil))

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		endpointHeadings,
		{
			"s3-gateway", "vpce-1", "Gateway", "Available",
			"com.amazonaws.us-east-1.s3", "us-east-1", "2024-01-02 03:04:05",
		},
	})
	uitest.OnUI(t, serviceCtx.App, func() {
		if len(api.vpcIds) != 1 || api.vpcIds[0] != "vpc-9" {
			t.Errorf("queried vpcs = %v, want [vpc-9]", api.vpcIds)
		}
	})
}

func TestSubnetsTable__RefreshLoadsSubnetsFromApi(t *testing.T) {
	var api = &fakeEc2Api{subnets: []types.Subnet{newSubnet("subnet-1", nameTag("private-a"))}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcSubnetsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshVpcSubnets(true, newVpc("vpc-1", nil))

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		subnetHeadings,
		{
			"private-a", "subnet-1", "available", "10.0.1.0/24", "251",
			"block-bidirectional", "us-east-1a", "123456789012", "private",
		},
	})
}

func TestSecurityGroupsTable__RefreshLoadsGroupsFromApi(t *testing.T) {
	var api = &fakeEc2Api{groups: []types.SecurityGroup{newSecurityGroup("sg-1", nameTag("web-sg"))}}
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewVpcSecurityGroupsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshVpcSecurityGroups(true, newVpc("vpc-1", nil))

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		securityGroupHeadings,
		{"web-sg", "sg-1", "web", "web servers", "2", "1", "123456789012"},
	})
}
