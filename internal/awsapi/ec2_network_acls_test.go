package awsapi

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// recordedForms keeps the form of each request, which the handler records
// from the goroutine the SDK sends on
type recordedForms struct {
	mtx   sync.Mutex
	forms []url.Values
}

func (r *recordedForms) add(body string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	var form, _ = url.ParseQuery(body)
	r.forms = append(r.forms, form)
}

func (r *recordedForms) get(index int) url.Values {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if index >= len(r.forms) {
		return nil
	}
	return r.forms[index]
}

func networkAclXml(id string) string {
	return `<item><networkAclId>` + id + `</networkAclId><vpcId>vpc-1</vpcId><default>false</default>` +
		`<entrySet><item><ruleNumber>100</ruleNumber><protocol>6</protocol><ruleAction>allow</ruleAction>` +
		`<egress>false</egress><cidrBlock>10.0.0.0/16</cidrBlock><portRange><from>443</from><to>443</to></portRange></item></entrySet>` +
		`<associationSet><item><networkAclAssociationId>aclassoc-1</networkAclAssociationId>` +
		`<networkAclId>` + id + `</networkAclId><subnetId>subnet-1</subnetId></item></associationSet></item>`
}

func TestEc2ListNetworkAclsListsEveryPageSortedById(t *testing.T) {
	var forms recordedForms
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		forms.add(body)
		if call == 1 {
			return response(200, "text/xml", `<DescribeNetworkAclsResponse><networkAclSet>`+
				networkAclXml("acl-2")+`</networkAclSet><nextToken>n1</nextToken></DescribeNetworkAclsResponse>`)
		}
		return response(200, "text/xml", `<DescribeNetworkAclsResponse><networkAclSet>`+
			networkAclXml("acl-1")+`</networkAclSet></DescribeNetworkAclsResponse>`)
	})

	var acls, err = NewEc2Api(fake.clients, testLogger()).ListNetworkAcls(context.Background(), true)

	if err != nil || len(acls) != 2 {
		t.Fatalf("acls = %v, err = %v, want two", acls, err)
	}
	if aws.ToString(acls[0].NetworkAclId) != "acl-1" || aws.ToString(acls[1].NetworkAclId) != "acl-2" {
		t.Errorf("acls = %s, %s, want them sorted by id", aws.ToString(acls[0].NetworkAclId), aws.ToString(acls[1].NetworkAclId))
	}
	var entry = acls[0].Entries[0]
	if aws.ToInt32(entry.RuleNumber) != 100 || aws.ToInt32(entry.PortRange.From) != 443 ||
		aws.ToString(acls[0].Associations[0].SubnetId) != "subnet-1" {
		t.Errorf("acl = %+v, want its rules and associations", acls[0])
	}
	if action := forms.get(0).Get("Action"); action != "DescribeNetworkAcls" {
		t.Errorf("action = %q", action)
	}
	if token := forms.get(1).Get("NextToken"); token != "n1" {
		t.Errorf("second page token = %q, want n1", token)
	}
}

func TestEc2ListNetworkAclsReturnsTheError(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return response(403, "text/xml", `<Response><Errors><Error><Code>UnauthorizedOperation</Code>`+
			`<Message>not allowed</Message></Error></Errors></Response>`)
	})

	var _, err = NewEc2Api(fake.clients, testLogger()).ListNetworkAcls(context.Background(), true)

	if err == nil {
		t.Error("err = nil, want the error")
	}
}
