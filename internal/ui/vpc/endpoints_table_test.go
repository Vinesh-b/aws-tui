package vpc

import (
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var endpointHeadings = core.TableRow{
	"Name", "Endpoint Id", "Endpoint Type", "Status", "Service Name", "Service Region", "Created Timestamp",
}

func newEndpoint(id string, tags []types.Tag) types.VpcEndpoint {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return types.VpcEndpoint{
		VpcEndpointId:     aws.String(id),
		VpcEndpointType:   types.VpcEndpointTypeGateway,
		State:             types.StateAvailable,
		ServiceName:       aws.String("com.amazonaws.us-east-1.s3"),
		ServiceRegion:     aws.String("us-east-1"),
		CreationTimestamp: &created,
		Tags:              tags,
	}
}

func newPopulatedEndpointsTable(endpoints ...types.VpcEndpoint) *VpcEndpointsTable {
	var table = NewVpcEndpointsTable(uitest.NewServiceContext[awsapi.Ec2Api]())
	table.populateVpcEndpointsTable(endpoints)
	return table
}

func TestEndpointsTable__ShowsHeadingsAndOneRowPerEndpoint(t *testing.T) {
	var table = newPopulatedEndpointsTable(newEndpoint("vpce-1", nameTag("s3-gateway")))

	uitest.AssertTableRows(t, table, []core.TableRow{
		endpointHeadings,
		{
			"s3-gateway", "vpce-1", "Gateway", "Available",
			"com.amazonaws.us-east-1.s3", "us-east-1", "2024-01-02 03:04:05",
		},
	})
}

func TestEndpointsTable__EmptyDataShowsOnlyHeadings(t *testing.T) {
	var table = newPopulatedEndpointsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{endpointHeadings})
}

func TestEndpointsTable__SelectingRowTracksSelectedEndpoint(t *testing.T) {
	var table = newPopulatedEndpointsTable(
		newEndpoint("vpce-1", nil),
		newEndpoint("vpce-2", nil),
	)

	table.Select(2, 0)

	if got := aws.ToString(table.GetSelectedVpcEndpoint().VpcEndpointId); got != "vpce-2" {
		t.Errorf("selected endpoint = %q, want %q", got, "vpce-2")
	}
}
