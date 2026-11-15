package awsapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func subnetXml(id string, vpcId string, cidr string) string {
	return `<item><subnetId>` + id + `</subnetId><vpcId>` + vpcId + `</vpcId><cidrBlock>` + cidr + `</cidrBlock>` +
		`<availabilityZone>us-east-1a</availabilityZone><state>available</state></item>`
}

func TestEc2ListSubnetsListsEveryPageOfEveryVpcSortedByVpcThenCidr(t *testing.T) {
	var forms recordedForms
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		forms.add(body)
		if call == 1 {
			return response(200, "text/xml", `<DescribeSubnetsResponse><subnetSet>`+
				subnetXml("subnet-3", "vpc-2", "10.1.0.0/24")+subnetXml("subnet-2", "vpc-1", "10.0.2.0/24")+
				`</subnetSet><nextToken>n1</nextToken></DescribeSubnetsResponse>`)
		}
		return response(200, "text/xml", `<DescribeSubnetsResponse><subnetSet>`+
			subnetXml("subnet-1", "vpc-1", "10.0.1.0/24")+`</subnetSet></DescribeSubnetsResponse>`)
	})

	var subnets, err = NewEc2Api(fake.clients, testLogger()).ListSubnets(context.Background(), true)

	if err != nil || len(subnets) != 3 {
		t.Fatalf("subnets = %v, err = %v, want three", subnets, err)
	}
	var ids []string
	for _, subnet := range subnets {
		ids = append(ids, aws.ToString(subnet.SubnetId))
	}
	if ids[0] != "subnet-1" || ids[1] != "subnet-2" || ids[2] != "subnet-3" {
		t.Errorf("subnets = %v, want them sorted by vpc then cidr", ids)
	}
	if action := forms.get(0).Get("Action"); action != "DescribeSubnets" {
		t.Errorf("action = %q", action)
	}
	if filter := forms.get(0).Get("Filter.1.Name"); filter != "" {
		t.Errorf("filter = %q, want the subnets of every vpc", filter)
	}
	if token := forms.get(1).Get("NextToken"); token != "n1" {
		t.Errorf("second page token = %q, want n1", token)
	}
}

func TestEc2ListSubnetsReturnsTheError(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(403, "text/xml", `<Response><Errors><Error><Code>UnauthorizedOperation</Code>`+
			`<Message>not allowed</Message></Error></Errors></Response>`)
	})

	var _, err = NewEc2Api(fake.clients, testLogger()).ListSubnets(context.Background(), true)

	if err == nil {
		t.Error("err = nil, want the error")
	}
}
