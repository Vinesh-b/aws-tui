package ec2

import (
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/ec2/ec2fake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var instanceHeadings = core.TableRow{
	"Name", "Instance ID", "State", "Type", "Availability Zone", "Private IP", "Public IP",
}

func newInstance(name string, instanceId string) types.Instance {
	var launched = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	var instance = types.Instance{
		InstanceId:       aws.String(instanceId),
		InstanceType:     types.InstanceTypeT3Micro,
		State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
		LaunchTime:       &launched,
		Placement:        &types.Placement{AvailabilityZone: aws.String("us-east-1a")},
		PrivateIpAddress: aws.String("10.0.0." + instanceId[len(instanceId)-1:]),
		VpcId:            aws.String("vpc-1"),
		SecurityGroups:   []types.GroupIdentifier{{GroupId: aws.String("sg-1"), GroupName: aws.String("web")}},
		BlockDeviceMappings: []types.InstanceBlockDeviceMapping{{
			DeviceName: aws.String("/dev/xvda"),
			Ebs:        &types.EbsInstanceBlockDevice{VolumeId: aws.String("vol-1"), Status: types.AttachmentStatusAttached},
		}},
		Tags: []types.Tag{{Key: aws.String("team"), Value: aws.String("payments")}},
	}
	if name != "" {
		instance.Tags = append(instance.Tags, types.Tag{Key: aws.String("Name"), Value: aws.String(name)})
	}
	return instance
}

func newApi() *ec2fake.Api {
	return ec2fake.New(ec2fake.WithInstances([]types.Instance{
		newInstance("web", "i-1"), newInstance("", "i-2"),
	}))
}

func TestInstancesTable__RefreshLoadsInstances(t *testing.T) {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](newApi())
	var table = NewEc2InstancesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshInstances(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		instanceHeadings,
		{"web", "i-1", "running", "t3.micro", "us-east-1a", "10.0.0.1", ""},
		{"", "i-2", "running", "t3.micro", "us-east-1a", "10.0.0.2", ""},
	})
}

func TestInstancesTable__RefreshReportsApiError(t *testing.T) {
	var api = ec2fake.New(ec2fake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewEc2InstancesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshInstances(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestInstancesTable__SearchFiltersTheInstancesWithoutListingThemAgain(t *testing.T) {
	var api = newApi()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Ec2Api](api)
	var table = NewEc2InstancesTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshInstances(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	// Setting the search text searches the instances, as when the search is done
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("i-2") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		instanceHeadings,
		{"", "i-2", "running", "t3.micro", "us-east-1a", "10.0.0.2", ""},
	})
	if got := api.InstanceListings(); got != 1 {
		t.Errorf("instances were listed %d times, want 1", got)
	}
}

func TestInstancesTable__SelectingRowTracksSelectedInstance(t *testing.T) {
	var table = NewEc2InstancesTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.data = []types.Instance{newInstance("web", "i-1"), newInstance("", "i-2")}
	table.populateInstancesTable()

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedInstance().InstanceId); got != "i-2" {
		t.Errorf("selected instance = %q, want i-2", got)
	}
}

func TestInstancesTable__EmptyTableHasNoSelectedInstance(t *testing.T) {
	var table = NewEc2InstancesTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.data = []types.Instance{newInstance("web", "i-1")}
	table.populateInstancesTable()
	table.Select(1, 0)

	table.data = nil
	table.populateInstancesTable()

	if got := aws.ToString(table.GetSelectedInstance().InstanceId); got != "" {
		t.Errorf("selected instance = %q, want none", got)
	}
}

func TestDetailsTable__ShowsTheInstance(t *testing.T) {
	var table = NewEc2DetailsTable(uitest.NewServiceContext[awsapi.Ec2Api]())

	table.SetInstance(newInstance("web", "i-1"))

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Name", "web"},
		{"Instance ID", "i-1"},
		{"State", "running"},
		{"Type", "t3.micro"},
		{"Launched", "2024-01-02 03:04:05"},
		{"Availability Zone", "us-east-1a"},
		{"Tenancy", ""},
		{"Architecture", ""},
		{"Platform", ""},
		{"AMI", ""},
		{"Key Pair", ""},
		{"IAM Instance Profile", ""},
		{"Monitoring", ""},
		{"Network"},
		{"VPC ID", "vpc-1"},
		{"Subnet ID", ""},
		{"Private IP", "10.0.0.1"},
		{"Private DNS", ""},
		{"Public IP", ""},
		{"Public DNS", ""},
		{"Security Groups"},
		{"sg-1", "web"},
		{"Volumes"},
		{"/dev/xvda", "vol-1 attached"},
		{"Tags"},
		{"Name", "web"},
		{"team", "payments"},
	})
}
