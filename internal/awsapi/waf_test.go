package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

const testWebAclArn = "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/main/a1b2"

func wafResponse(body string) *http.Response {
	return response(200, "application/x-amz-json-1.1", body)
}

func wafTarget(req *http.Request) string {
	return req.Header.Get("X-Amz-Target")
}

func TestWafListWebACLsAsksForRegionalOnesPageByPageSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch call {
		case 1:
			return wafResponse(`{"WebACLs":[{"Name":"orders"}],"NextMarker":"m1"}`)
		case 2:
			return wafResponse(`{"WebACLs":[{"Name":"api"}],"NextMarker":"m2"}`)
		}
		// WAF can give a marker with the last page, which is followed by an
		// empty one
		return wafResponse(`{"WebACLs":[],"NextMarker":"m3"}`)
	})

	var acls, err = NewWafApi(fake.clients, testLogger()).ListWebACLs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(acls) != 2 || aws.ToString(acls[0].Name) != "api" || aws.ToString(acls[1].Name) != "orders" {
		t.Errorf("web ACLs = %v, want [api orders]", acls)
	}
	if fake.calls() != 3 {
		t.Errorf("calls = %d, want 3", fake.calls())
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"Scope":"REGIONAL"`) {
		t.Errorf("first request = %s, want the regional scope", body)
	}
	if body := fake.requestBody(1); !strings.Contains(body, `"NextMarker":"m1"`) {
		t.Errorf("second request = %s, want the marker of the first page", body)
	}
}

func TestWafGetWebACLAsksForItByArn(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return wafResponse(`{"WebACL":{"Name":"main","Capacity":700,"Rules":[{"Name":"block-bad-ips","Priority":1}]}}`)
	})

	var acl, err = NewWafApi(fake.clients, testLogger()).GetWebACL(context.Background(), testWebAclArn)
	if err != nil {
		t.Fatal(err)
	}
	if acl.Capacity != 700 || len(acl.Rules) != 1 || aws.ToString(acl.Rules[0].Name) != "block-bad-ips" {
		t.Errorf("web ACL = %+v, want main with its rule", acl)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"ARN":"`+testWebAclArn+`"`) {
		t.Errorf("request = %s, want the ARN", body)
	}
}

func TestWafListWebACLResourcesAsksForEachRegionalType(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		switch {
		case strings.Contains(body, `"ResourceType":"APPLICATION_LOAD_BALANCER"`):
			return wafResponse(`{"ResourceArns":["arn:alb/web-b","arn:alb/web-a"]}`)
		case strings.Contains(body, `"ResourceType":"API_GATEWAY"`):
			return wafResponse(`{"ResourceArns":["arn:apigateway/stage"]}`)
		}
		return wafResponse(`{"ResourceArns":[]}`)
	})

	var resources, err = NewWafApi(fake.clients, testLogger()).ListWebACLResources(context.Background(), testWebAclArn)
	if err != nil {
		t.Fatal(err)
	}

	var want = []WebACLResource{
		{types.ResourceTypeApplicationLoadBalancer, "arn:alb/web-a"},
		{types.ResourceTypeApplicationLoadBalancer, "arn:alb/web-b"},
		{types.ResourceTypeApiGateway, "arn:apigateway/stage"},
	}
	if len(resources) != len(want) {
		t.Fatalf("resources = %v, want %v", resources, want)
	}
	for i := range want {
		if resources[i] != want[i] {
			t.Errorf("resource %d = %v, want %v", i, resources[i], want[i])
		}
	}
	if fake.calls() != len(WafRegionalResourceTypes) {
		t.Errorf("calls = %d, want one for each of the %d types", fake.calls(), len(WafRegionalResourceTypes))
	}
	for i := range fake.calls() {
		if body := fake.requestBody(i); !strings.Contains(body, `"WebACLArn":"`+testWebAclArn+`"`) {
			t.Errorf("request %d = %s, want the web ACL ARN", i, body)
		}
	}
}

func TestWafListWebACLResourcesSkipsTypesTheRegionRejects(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if strings.Contains(body, `"ResourceType":"APP_RUNNER_SERVICE"`) {
			return response(400, "application/x-amz-json-1.1",
				`{"__type":"WAFInvalidParameterException","Message":"not supported in this region"}`)
		}
		if strings.Contains(body, `"ResourceType":"VERIFIED_ACCESS_INSTANCE"`) {
			return wafResponse(`{"ResourceArns":["arn:verified-access/instance"]}`)
		}
		return wafResponse(`{"ResourceArns":[]}`)
	})

	var resources, err = NewWafApi(fake.clients, testLogger()).ListWebACLResources(context.Background(), testWebAclArn)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Arn != "arn:verified-access/instance" {
		t.Errorf("resources = %v, want the verified access instance", resources)
	}
}

func TestWafListWebACLResourcesStopsAtOtherErrors(t *testing.T) {
	var fake = useFakeAws(t, func(call int, req *http.Request, body string) *http.Response {
		if strings.Contains(body, `"ResourceType":"API_GATEWAY"`) {
			return response(400, "application/x-amz-json-1.1",
				`{"__type":"AccessDeniedException","Message":"not allowed"}`)
		}
		return wafResponse(`{"ResourceArns":["arn:alb/web"]}`)
	})

	var resources, err = NewWafApi(fake.clients, testLogger()).ListWebACLResources(context.Background(), testWebAclArn)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("err = %v, want the access denied error", err)
	}
	if len(resources) != 1 || resources[0].Arn != "arn:alb/web" {
		t.Errorf("resources = %v, want those found before the error", resources)
	}
	if fake.calls() != 2 {
		t.Errorf("calls = %d, want none after the error", fake.calls())
	}
}

func TestWafListIPSetsAsksForRegionalOnesSortedByName(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return wafResponse(`{"IPSets":[{"Name":"office"},{"Name":"blocklist"}]}`)
	})

	var sets, err = NewWafApi(fake.clients, testLogger()).ListIPSets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 2 || aws.ToString(sets[0].Name) != "blocklist" {
		t.Errorf("IP sets = %v, want [blocklist office]", sets)
	}
	if body := fake.requestBody(0); !strings.Contains(body, `"Scope":"REGIONAL"`) {
		t.Errorf("request = %s, want the regional scope", body)
	}
}

func TestWafGetIPSetAsksForTheNameIdAndScopeOfTheArn(t *testing.T) {
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		if !strings.HasSuffix(wafTarget(req), ".GetIPSet") {
			t.Errorf("target = %s, want GetIPSet", wafTarget(req))
		}
		return wafResponse(`{"IPSet":{"Name":"blocklist","Addresses":["192.0.2.0/24"],"IPAddressVersion":"IPV4"}}`)
	})

	var set, err = NewWafApi(fake.clients, testLogger()).GetIPSet(context.Background(),
		"arn:aws:wafv2:us-east-1:123456789012:regional/ipset/blocklist/c3d4")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Addresses) != 1 || set.IPAddressVersion != types.IPAddressVersionIpv4 {
		t.Errorf("IP set = %+v, want its address", set)
	}
	var body = fake.requestBody(0)
	for _, want := range []string{`"Name":"blocklist"`, `"Id":"c3d4"`, `"Scope":"REGIONAL"`} {
		if !strings.Contains(body, want) {
			t.Errorf("request = %s, want %s", body, want)
		}
	}
}

func TestWafGetIPSetRejectsAnArnThatIsNotOfWaf(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return wafResponse(`{}`)
	})

	var _, err = NewWafApi(fake.clients, testLogger()).GetIPSet(context.Background(),
		"arn:aws:s3:us-east-1:123456789012:accesspoint/reports")
	if err == nil || fake.calls() != 0 {
		t.Errorf("err = %v with %d calls, want an error and no request", err, fake.calls())
	}
}

func TestParseWafArn(t *testing.T) {
	for _, test := range []struct {
		arn     string
		want    WafArn
		wantErr bool
	}{
		{testWebAclArn, WafArn{types.ScopeRegional, "webacl", "main", "a1b2"}, false},
		{"arn:aws:wafv2:us-east-1:123456789012:global/ipset/office/e5f6", WafArn{types.ScopeCloudfront, "ipset", "office", "e5f6"}, false},
		{"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/main", WafArn{}, true},
		{"arn:aws:wafv2:us-east-1:123456789012:regional/webacl//a1b2", WafArn{}, true},
		{"arn:aws:wafv2:us-east-1:123456789012:other/webacl/main/a1b2", WafArn{}, true},
		{"arn:aws:waf-regional:us-east-1:123456789012:webacl/a1b2", WafArn{}, true},
		{"not an arn", WafArn{}, true},
	} {
		var got, err = ParseWafArn(test.arn)
		if (err != nil) != test.wantErr || got != test.want {
			t.Errorf("ParseWafArn(%q) = %+v, %v, want %+v, error %t", test.arn, got, err, test.want, test.wantErr)
		}
	}
}
