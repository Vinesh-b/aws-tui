package awsapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"aws-tui/internal/safetest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

const route53Ns = `xmlns="https://route53.amazonaws.com/doc/2013-04-01/"`

func route53Response(body string) *http.Response {
	return response(200, "text/xml", body)
}

// requestUrl is the path and query of the request
func requestUrl(req *http.Request) string {
	return req.URL.RequestURI()
}

func TestRoute53ListHostedZonesFollowsTheMarkerSortedByName(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(call int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		if call == 1 {
			return route53Response(`<ListHostedZonesResponse ` + route53Ns + `><HostedZones>` +
				`<HostedZone><Id>/hostedzone/Z2</Id><Name>shop.example.com.</Name></HostedZone>` +
				`</HostedZones><IsTruncated>true</IsTruncated><NextMarker>Z1</NextMarker><MaxItems>1</MaxItems>` +
				`</ListHostedZonesResponse>`)
		}
		return route53Response(`<ListHostedZonesResponse ` + route53Ns + `><HostedZones>` +
			`<HostedZone><Id>/hostedzone/Z1</Id><Name>api.example.com.</Name>` +
			`<Config><PrivateZone>true</PrivateZone></Config></HostedZone>` +
			`</HostedZones><IsTruncated>false</IsTruncated><MaxItems>1</MaxItems></ListHostedZonesResponse>`)
	})

	var zones, err = NewRoute53Api(fake.clients, testLogger()).ListHostedZones(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(zones) != 2 || aws.ToString(zones[0].Name) != "api.example.com." || !zones[0].Config.PrivateZone {
		t.Errorf("zones = %v, want the private api zone then shop", zones)
	}
	if got := urls.Calls(); len(got) != 2 || !strings.Contains(got[1], "marker=Z1") {
		t.Errorf("requests = %v, want the second to have the marker of the first", got)
	}
}

func TestRoute53GetHostedZoneGivesItsNameServersAndVPCs(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		return route53Response(`<GetHostedZoneResponse ` + route53Ns + `>` +
			`<HostedZone><Id>/hostedzone/Z1</Id><Name>example.com.</Name><ResourceRecordSetCount>12</ResourceRecordSetCount></HostedZone>` +
			`<DelegationSet><NameServers><NameServer>ns-1.awsdns-01.org</NameServer><NameServer>ns-2.awsdns-02.com</NameServer></NameServers></DelegationSet>` +
			`<VPCs><VPC><VPCRegion>eu-west-1</VPCRegion><VPCId>vpc-0abc</VPCId></VPC></VPCs>` +
			`</GetHostedZoneResponse>`)
	})

	// The id is given as the list has it
	var zone, err = NewRoute53Api(fake.clients, testLogger()).GetHostedZone(context.Background(), "/hostedzone/Z1")
	if err != nil {
		t.Fatal(err)
	}
	if aws.ToInt64(zone.Zone.ResourceRecordSetCount) != 12 || len(zone.NameServers) != 2 ||
		len(zone.VPCs) != 1 || aws.ToString(zone.VPCs[0].VPCId) != "vpc-0abc" {
		t.Errorf("zone = %+v, want its record count, name servers and VPC", zone)
	}
	if got := urls.Calls(); len(got) != 1 || got[0] != "/2013-04-01/hostedzone/Z1" {
		t.Errorf("requests = %v, want the zone by its bare id", got)
	}
}

func TestRoute53ListRecordsFollowsTheNextRecord(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(call int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		if call == 1 {
			return route53Response(`<ListResourceRecordSetsResponse ` + route53Ns + `><ResourceRecordSets>` +
				`<ResourceRecordSet><Name>example.com.</Name><Type>A</Type><TTL>300</TTL>` +
				`<ResourceRecords><ResourceRecord><Value>192.0.2.1</Value></ResourceRecord></ResourceRecords></ResourceRecordSet>` +
				`</ResourceRecordSets><IsTruncated>true</IsTruncated><NextRecordName>www.example.com.</NextRecordName>` +
				`<NextRecordType>CNAME</NextRecordType><MaxItems>1</MaxItems></ListResourceRecordSetsResponse>`)
		}
		return route53Response(`<ListResourceRecordSetsResponse ` + route53Ns + `><ResourceRecordSets>` +
			`<ResourceRecordSet><Name>www.example.com.</Name><Type>CNAME</Type>` +
			`<AliasTarget><HostedZoneId>Z2</HostedZoneId><DNSName>shop.example.net.</DNSName>` +
			`<EvaluateTargetHealth>false</EvaluateTargetHealth></AliasTarget></ResourceRecordSet>` +
			`</ResourceRecordSets><IsTruncated>false</IsTruncated><MaxItems>1</MaxItems></ListResourceRecordSetsResponse>`)
	})

	var records, err = NewRoute53Api(fake.clients, testLogger()).ListRecords(context.Background(), "/hostedzone/Z1")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Type != types.RRTypeA ||
		aws.ToString(records[1].AliasTarget.DNSName) != "shop.example.net." {
		t.Errorf("records = %+v, want the A record then the alias", records)
	}
	var got = urls.Calls()
	if len(got) != 2 || !strings.HasPrefix(got[0], "/2013-04-01/hostedzone/Z1/rrset") ||
		!strings.Contains(got[1], "name=www.example.com.") || !strings.Contains(got[1], "type=CNAME") {
		t.Errorf("requests = %v, want the zone by its bare id, then from the next record", got)
	}
}

func TestRoute53ListRecordsGivesThoseFoundBeforeAnError(t *testing.T) {
	var fake = useFakeAws(t, func(call int, _ *http.Request, _ string) *http.Response {
		if call == 1 {
			return route53Response(`<ListResourceRecordSetsResponse ` + route53Ns + `><ResourceRecordSets>` +
				`<ResourceRecordSet><Name>example.com.</Name><Type>NS</Type></ResourceRecordSet>` +
				`</ResourceRecordSets><IsTruncated>true</IsTruncated><NextRecordName>example.com.</NextRecordName>` +
				`<NextRecordType>SOA</NextRecordType><MaxItems>1</MaxItems></ListResourceRecordSetsResponse>`)
		}
		return response(400, "text/xml", `<ErrorResponse `+route53Ns+`><Error><Type>Sender</Type>`+
			`<Code>Throttling</Code><Message>Rate exceeded</Message></Error></ErrorResponse>`)
	})

	var records, err = NewRoute53Api(fake.clients, testLogger()).ListRecords(context.Background(), "Z1")
	if err == nil || !strings.Contains(err.Error(), "Rate exceeded") {
		t.Errorf("err = %v, want the throttling error", err)
	}
	if len(records) != 1 || records[0].Type != types.RRTypeNs {
		t.Errorf("records = %v, want the one found before the error", records)
	}
}

func TestRoute53ListHealthChecksSortedById(t *testing.T) {
	var fake = useFakeAws(t, func(int, *http.Request, string) *http.Response {
		return route53Response(`<ListHealthChecksResponse ` + route53Ns + `><HealthChecks>` +
			`<HealthCheck><Id>b2</Id><HealthCheckConfig><Type>TCP</Type></HealthCheckConfig></HealthCheck>` +
			`<HealthCheck><Id>a1</Id><HealthCheckConfig><Type>HTTPS</Type><Port>443</Port></HealthCheckConfig></HealthCheck>` +
			`</HealthChecks><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems><Marker></Marker></ListHealthChecksResponse>`)
	})

	var checks, err = NewRoute53Api(fake.clients, testLogger()).ListHealthChecks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 || aws.ToString(checks[0].Id) != "a1" || checks[0].HealthCheckConfig.Type != types.HealthCheckTypeHttps {
		t.Errorf("health checks = %v, want [a1 b2]", checks)
	}
}

func TestRoute53GetHealthCheckStatusSortsByRegionThenAddress(t *testing.T) {
	var urls safetest.Recorder[string]
	var fake = useFakeAws(t, func(_ int, req *http.Request, _ string) *http.Response {
		urls.Record(requestUrl(req))
		return route53Response(`<GetHealthCheckStatusResponse ` + route53Ns + `><HealthCheckObservations>` +
			`<HealthCheckObservation><Region>us-west-1</Region><IPAddress>15.177.2.1</IPAddress>` +
			`<StatusReport><Status>Success: HTTP Status Code 200, OK</Status></StatusReport></HealthCheckObservation>` +
			`<HealthCheckObservation><Region>eu-west-1</Region><IPAddress>15.177.62.9</IPAddress>` +
			`<StatusReport><Status>Failure: Connection timed out</Status></StatusReport></HealthCheckObservation>` +
			`<HealthCheckObservation><Region>eu-west-1</Region><IPAddress>15.177.62.1</IPAddress>` +
			`<StatusReport><Status>Success: HTTP Status Code 200, OK</Status></StatusReport></HealthCheckObservation>` +
			`</HealthCheckObservations></GetHealthCheckStatusResponse>`)
	})

	var observations, err = NewRoute53Api(fake.clients, testLogger()).GetHealthCheckStatus(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	var addresses []string
	for _, observation := range observations {
		addresses = append(addresses, aws.ToString(observation.IPAddress))
	}
	if strings.Join(addresses, " ") != "15.177.62.1 15.177.62.9 15.177.2.1" {
		t.Errorf("observations are from %v, want eu-west-1 by address then us-west-1", addresses)
	}
	if got := urls.Calls(); len(got) != 1 || got[0] != "/2013-04-01/healthcheck/a1/status" {
		t.Errorf("requests = %v, want the status of a1", got)
	}
}

func TestParseRoute53Arn(t *testing.T) {
	for _, test := range []struct {
		arn     string
		want    Route53Arn
		wantErr bool
	}{
		{"arn:aws:route53:::hostedzone/Z0123456789ABC", Route53Arn{"hostedzone", "Z0123456789ABC"}, false},
		{"arn:aws:route53:::healthcheck/abcd-1234", Route53Arn{"healthcheck", "abcd-1234"}, false},
		{"arn:aws:route53:::hostedzone/", Route53Arn{}, true},
		{"arn:aws:route53:::hostedzone", Route53Arn{}, true},
		{"arn:aws:route53:::hostedzone/Z1/extra", Route53Arn{}, true},
		{"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/main/a1b2", Route53Arn{}, true},
		{"not an arn", Route53Arn{}, true},
	} {
		var got, err = ParseRoute53Arn(test.arn)
		if (err != nil) != test.wantErr || got != test.want {
			t.Errorf("ParseRoute53Arn(%q) = %+v, %v, want %+v, error %t", test.arn, got, err, test.want, test.wantErr)
		}
	}
}

func TestRoute53ZoneId(t *testing.T) {
	for id, want := range map[string]string{
		"/hostedzone/Z0123456789ABC": "Z0123456789ABC",
		"Z0123456789ABC":             "Z0123456789ABC",
		"":                           "",
	} {
		if got := Route53ZoneId(id); got != want {
			t.Errorf("Route53ZoneId(%q) = %q, want %q", id, got, want)
		}
	}
}
