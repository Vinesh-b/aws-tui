package route53

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/route53/route53fake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var zoneHeadings = core.TableRow{"Name", "Type", "Records", "Comment", "Id"}
var recordHeadings = core.TableRow{"Name", "Type", "Routing", "Record Id", "TTL", "Value"}
var healthCheckHeadings = core.TableRow{"Id", "Type", "Target"}
var statusHeadings = core.TableRow{"Region", "IP Address", "Checked", "Status"}

// zoneIdOf is the id of the zone of the name, as the list gives it
func zoneIdOf(name string) string {
	return "/hostedzone/Z" + strings.ToUpper(strings.ReplaceAll(name, ".", ""))
}

func newZone(name string, records int64) types.HostedZone {
	return types.HostedZone{
		Id:                     aws.String(zoneIdOf(name)),
		Name:                   aws.String(name + "."),
		ResourceRecordSetCount: aws.Int64(records),
		Config:                 &types.HostedZoneConfig{Comment: aws.String("Zone of " + name)},
	}
}

func newPrivateZone(name string, records int64) types.HostedZone {
	var zone = newZone(name, records)
	zone.Config.PrivateZone = true
	return zone
}

func simpleRecord(name string, recordType types.RRType, values ...string) types.ResourceRecordSet {
	var record = types.ResourceRecordSet{
		Name: aws.String(name),
		Type: recordType,
		TTL:  aws.Int64(300),
	}
	for _, value := range values {
		record.ResourceRecords = append(record.ResourceRecords, types.ResourceRecord{Value: aws.String(value)})
	}
	return record
}

func httpsCheck(id string, host string) types.HealthCheck {
	return types.HealthCheck{
		Id:                 aws.String(id),
		HealthCheckVersion: aws.Int64(3),
		HealthCheckConfig: &types.HealthCheckConfig{
			Type:                     types.HealthCheckTypeHttps,
			FullyQualifiedDomainName: aws.String(host),
			Port:                     aws.Int32(443),
			ResourcePath:             aws.String("/health"),
			RequestInterval:          aws.Int32(30),
			FailureThreshold:         aws.Int32(3),
			MeasureLatency:           aws.Bool(false),
			Inverted:                 aws.Bool(false),
			Disabled:                 aws.Bool(false),
			EnableSNI:                aws.Bool(true),
			Regions:                  []types.HealthCheckRegion{types.HealthCheckRegionEuWest1, types.HealthCheckRegionUsEast1},
		},
	}
}

func calculatedCheck(id string, threshold int32, children ...string) types.HealthCheck {
	return types.HealthCheck{
		Id:                 aws.String(id),
		HealthCheckVersion: aws.Int64(1),
		HealthCheckConfig: &types.HealthCheckConfig{
			Type:              types.HealthCheckTypeCalculated,
			HealthThreshold:   aws.Int32(threshold),
			ChildHealthChecks: children,
			Inverted:          aws.Bool(false),
			Disabled:          aws.Bool(false),
		},
	}
}

func observation(region types.HealthCheckRegion, ip string, status string) types.HealthCheckObservation {
	return types.HealthCheckObservation{
		Region:    region,
		IPAddress: aws.String(ip),
		StatusReport: &types.StatusReport{
			Status:      aws.String(status),
			CheckedTime: aws.Time(time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)),
		},
	}
}

func TestDnsName(t *testing.T) {
	for name, want := range map[string]string{
		"example.com.":            "example.com",
		`\052.example.com.`:       "*.example.com",
		`a\100b.example.com.`:     "a@b.example.com",
		`not\9escape.example.com`: `not\9escape.example.com`,
		`trailing\05`:             `trailing\05`,
		`\400.example.com.`:       `\400.example.com`,
		".":                       ".",
		"":                        "",
	} {
		if got := dnsName(name); got != want {
			t.Errorf("dnsName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRoutingPolicy(t *testing.T) {
	for _, test := range []struct {
		name   string
		record types.ResourceRecordSet
		want   string
	}{
		{"simple", types.ResourceRecordSet{}, "Simple"},
		{"weighted at zero", types.ResourceRecordSet{Weight: aws.Int64(0)}, "Weighted 0"},
		{"latency", types.ResourceRecordSet{Region: types.ResourceRecordSetRegionEuWest1}, "Latency eu-west-1"},
		{"failover", types.ResourceRecordSet{Failover: types.ResourceRecordSetFailoverSecondary}, "Failover secondary"},
		{"geolocation country", types.ResourceRecordSet{GeoLocation: &types.GeoLocation{CountryCode: aws.String("DE")}}, "Geolocation DE"},
		{"geolocation continent", types.ResourceRecordSet{GeoLocation: &types.GeoLocation{ContinentCode: aws.String("EU")}}, "Geolocation EU"},
		{"geolocation subdivision", types.ResourceRecordSet{GeoLocation: &types.GeoLocation{
			CountryCode: aws.String("US"), SubdivisionCode: aws.String("WA"),
		}}, "Geolocation US-WA"},
		{"geolocation default", types.ResourceRecordSet{GeoLocation: &types.GeoLocation{CountryCode: aws.String("*")}}, "Geolocation default"},
		{"geoproximity region", types.ResourceRecordSet{GeoProximityLocation: &types.GeoProximityLocation{
			AWSRegion: aws.String("eu-central-1"), Bias: aws.Int32(-20),
		}}, "Geoproximity eu-central-1 bias -20"},
		{"geoproximity coordinates", types.ResourceRecordSet{GeoProximityLocation: &types.GeoProximityLocation{
			Coordinates: &types.Coordinates{Latitude: aws.String("49.22"), Longitude: aws.String("-74.01")},
		}}, "Geoproximity 49.22,-74.01"},
		{"IP-based", types.ResourceRecordSet{CidrRoutingConfig: &types.CidrRoutingConfig{LocationName: aws.String("office")}}, "IP-based office"},
		{"multivalue", types.ResourceRecordSet{MultiValueAnswer: aws.Bool(true)}, "Multivalue"},
	} {
		if got := routingPolicy(test.record); got != test.want {
			t.Errorf("%s: routingPolicy = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestHealthCheckTarget(t *testing.T) {
	for _, test := range []struct {
		name   string
		config *types.HealthCheckConfig
		want   string
	}{
		{"none", nil, ""},
		{"domain name", &types.HealthCheckConfig{
			Type: types.HealthCheckTypeHttps, FullyQualifiedDomainName: aws.String("api.example.com"),
			Port: aws.Int32(443), ResourcePath: aws.String("/health"),
		}, "api.example.com:443/health"},
		{"IP address, with the domain name only the host", &types.HealthCheckConfig{
			Type: types.HealthCheckTypeHttp, FullyQualifiedDomainName: aws.String("api.example.com"),
			IPAddress: aws.String("192.0.2.10"), Port: aws.Int32(80),
		}, "192.0.2.10:80"},
		{"IPv6", &types.HealthCheckConfig{
			Type: types.HealthCheckTypeTcp, IPAddress: aws.String("2001:db8::1"), Port: aws.Int32(5432),
		}, "[2001:db8::1]:5432"},
		{"calculated", &types.HealthCheckConfig{
			Type: types.HealthCheckTypeCalculated, HealthThreshold: aws.Int32(2), ChildHealthChecks: []string{"a", "b", "c"},
		}, "2 of 3 child health checks"},
		{"alarm", &types.HealthCheckConfig{
			Type:            types.HealthCheckTypeCloudwatchMetric,
			AlarmIdentifier: &types.AlarmIdentifier{Name: aws.String("api-errors"), Region: types.CloudWatchRegionEuWest1},
		}, "Alarm api-errors in eu-west-1"},
	} {
		if got := healthCheckTarget(test.config); got != test.want {
			t.Errorf("%s: healthCheckTarget = %q, want %q", test.name, got, test.want)
		}
	}
}

// awaitDetailsLoaded waits for the details to have the row only loading the
// hosted zone gives, as AwaitDetailsRows is already satisfied by the rows from
// the list
func awaitDetailsLoaded(t *testing.T, app *tview.Application, table *core.DetailsTable) {
	t.Helper()
	uitest.Eventually(t, app, func() bool { return uitest.DetailsRowOf(table, "Name Servers") >= 0 },
		"the hosted zone to be loaded")
}

func newHostedZonesTest(t *testing.T, api *route53fake.Api) *HostedZonesPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Route53Api](api)
	var view = NewHostedZonesPageView(
		NewHostedZonesTable(serviceCtx),
		NewHostedZoneDetailsTable(serviceCtx),
		NewRecordsTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadZones loads the hosted zones of the page and waits for them
func loadZones(t *testing.T, view *HostedZonesPageView, count int) {
	t.Helper()
	var app = view.HostedZonesTable.serviceCtx.App
	view.HostedZonesTable.RefreshHostedZones()
	uitest.Eventually(t, app, func() bool {
		return view.HostedZonesTable.GetTable().GetRowCount() == count+1
	}, "the hosted zones to be loaded")
}

// switchTab moves from the details tab to the next one, and checks it is the
// one named
func switchTab(t *testing.T, app *tview.Application, details tview.Primitive, tabView *core.TabViewHorizontal, want string) {
	t.Helper()
	uitest.OnUI(t, app, func() { app.SetFocus(details) })
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.OnUI(t, app, func() {
		if name, _ := tabView.GetCurrentTab(); name != want {
			t.Fatalf("tab = %q, want %q", name, want)
		}
	})
}

func openRecordsTab(t *testing.T, view *HostedZonesPageView) {
	t.Helper()
	switchTab(t, view.HostedZonesTable.serviceCtx.App, view.DetailsTable, view.TabView, HostedZoneTabNameRecords)
}

func TestHostedZonesTable__ResetLoadsTheHostedZones(t *testing.T) {
	var api = route53fake.New(route53fake.WithHostedZones([]types.HostedZone{
		newZone("example.com", 14), newPrivateZone("internal.example.com", 3),
	}))
	var view = newHostedZonesTest(t, api)
	var app = view.HostedZonesTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.HostedZonesTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.HostedZonesTable, []core.TableRow{
		zoneHeadings,
		{"example.com", "Public", "14", "Zone of example.com", "ZEXAMPLECOM"},
		{"internal.example.com", "Private", "3", "Zone of internal.example.com", "ZINTERNALEXAMPLECOM"},
	})
}

func TestHostedZonesTable__ErrorIsShown(t *testing.T) {
	var api = route53fake.New(route53fake.WithErr(errors.New("AccessDenied: route53:ListHostedZones")))
	var view = newHostedZonesTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.HostedZonesTable.serviceCtx.App, func() { view.HostedZonesTable.ErrorMessageCallback = callback })

	view.HostedZonesTable.RefreshHostedZones()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "route53:ListHostedZones") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestHostedZonesPage__SelectingAZoneShowsItThenItsNameServersAndVPCs(t *testing.T) {
	var zone = newPrivateZone("internal.example.com", 3)
	var api = route53fake.New(
		route53fake.WithHostedZones([]types.HostedZone{zone}),
		route53fake.WithHostedZone(zoneIdOf("internal.example.com"), awsapi.HostedZoneDetails{
			Zone:        zone,
			NameServers: []string{"ns-1.awsdns-01.org", "ns-2.awsdns-02.com"},
			VPCs: []types.VPC{
				{VPCId: aws.String("vpc-0abc"), VPCRegion: types.VPCRegionEuWest1},
				{VPCId: aws.String("vpc-0def"), VPCRegion: types.VPCRegionUsEast1},
			},
		}),
	)
	var view = newHostedZonesTest(t, api)
	var app = view.HostedZonesTable.serviceCtx.App

	view.HostedZonesTable.RefreshHostedZones()

	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable)
	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "internal.example.com"},
		{"Id", "ZINTERNALEXAMPLECOM"},
		{"ARN", "arn:aws:route53:::hostedzone/ZINTERNALEXAMPLECOM"},
		{"Type", "Private"},
		{"Records", "3"},
		{"Comment", "Zone of internal.example.com"},
		{"Linked Service", ""},
		{"Name Servers", "ns-1.awsdns-01.org, ns-2.awsdns-02.com"},
		{"VPCs"},
		{"vpc-0abc", "eu-west-1"},
		{"vpc-0def", "us-east-1"},
	})
}

func TestHostedZonesPage__MovingThroughZonesLoadsOnlyTheOneItStopsOn(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var names = []string{"a.example.com", "b.example.com", "c.example.com", "d.example.com"}
		var zones []types.HostedZone
		for _, name := range names {
			zones = append(zones, newZone(name, 2))
		}
		var api = route53fake.New(route53fake.WithHostedZones(zones))
		var view = newHostedZonesTest(t, api)
		loadZones(t, view, len(names))

		uitest.AssertMovingThroughLoadsOnlyWhereItStops(t, view.HostedZonesTable.serviceCtx.App, uitest.MoveThrough[string]{
			Table: view.HostedZonesTable, Passed: zoneIdOf("c.example.com"), StopsOn: zoneIdOf("d.example.com"),
			Loaded: api.ZonesAskedFor,
			// The zone is shown straight away, before it is loaded
			StraightAway: func() {
				if got := view.DetailsTable.GetCellText(0, 1); got != "d.example.com" {
					t.Errorf("zone shown = %q, want d.example.com", got)
				}
			},
		})
	})
}

func TestHostedZonesPage__RecordsAreOnlyLoadedOnceTheirTabIsOpened(t *testing.T) {
	var weighted = simpleRecord("api.example.com.", types.RRTypeA, "192.0.2.10")
	weighted.Weight = aws.Int64(80)
	weighted.SetIdentifier = aws.String("blue")
	var api = route53fake.New(
		route53fake.WithHostedZones([]types.HostedZone{newZone("example.com", 4)}),
		route53fake.WithRecords(zoneIdOf("example.com"), []types.ResourceRecordSet{
			simpleRecord("example.com.", types.RRTypeNs, "ns-1.awsdns-01.org.", "ns-2.awsdns-02.com."),
			weighted,
			{
				Name:        aws.String("www.example.com."),
				Type:        types.RRTypeA,
				AliasTarget: &types.AliasTarget{DNSName: aws.String("dualstack.shop-123.eu-west-1.elb.amazonaws.com.")},
			},
			simpleRecord(`\052.example.com.`, types.RRTypeCname, "example.com"),
		}),
	)
	var view = newHostedZonesTest(t, api)
	var app = view.HostedZonesTable.serviceCtx.App
	loadZones(t, view, 1)
	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable)
	if asked := api.RecordsAskedFor(); len(asked) != 0 {
		t.Fatalf("records loaded before the tab was opened: %v", asked)
	}

	openRecordsTab(t, view)

	uitest.AwaitRows(t, app, view.RecordsTable, []core.TableRow{
		recordHeadings,
		{"example.com", "NS", "Simple", "", "300", "ns-1.awsdns-01.org., ns-2.awsdns-02.com."},
		{"api.example.com", "A", "Weighted 80", "blue", "300", "192.0.2.10"},
		{"www.example.com", "A", "Simple", "", "", "ALIAS dualstack.shop-123.eu-west-1.elb.amazonaws.com."},
		{"*.example.com", "CNAME", "Simple", "", "300", "example.com"},
	})
	// Going round the tabs back to it doesn't load them again
	for range 2 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	uitest.OnUI(t, app, func() {
		if name, _ := view.TabView.GetCurrentTab(); name != HostedZoneTabNameRecords {
			t.Fatalf("tab = %q, want %q", name, HostedZoneTabNameRecords)
		}
	})
	if asked := api.RecordsAskedFor(); len(asked) != 1 {
		t.Errorf("records loaded after reopening the tab = %v, want example.com once", asked)
	}
}

func TestHostedZonesPage__SelectingAnotherZoneWithTheRecordsOpenLoadsItsRecords(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = route53fake.New(
			route53fake.WithHostedZones([]types.HostedZone{newZone("example.com", 1), newZone("example.net", 1)}),
			route53fake.WithRecords(zoneIdOf("example.com"), []types.ResourceRecordSet{
				simpleRecord("example.com.", types.RRTypeA, "192.0.2.1"),
			}),
			route53fake.WithRecords(zoneIdOf("example.net"), []types.ResourceRecordSet{
				simpleRecord("example.net.", types.RRTypeA, "198.51.100.1"),
			}),
		)
		var view = newHostedZonesTest(t, api)
		var app = view.HostedZonesTable.serviceCtx.App
		loadZones(t, view, 2)
		openRecordsTab(t, view)
		uitest.AwaitRows(t, app, view.RecordsTable, []core.TableRow{
			recordHeadings, {"example.com", "A", "Simple", "", "300", "192.0.2.1"},
		})

		uitest.OnUI(t, app, func() { view.HostedZonesTable.Select(2, 0) })

		uitest.AwaitRows(t, app, view.RecordsTable, []core.TableRow{
			recordHeadings, {"example.net", "A", "Simple", "", "300", "198.51.100.1"},
		})
	})
}

func TestHostedZonesPage__RecordsErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = route53fake.New(
		route53fake.WithHostedZones([]types.HostedZone{newZone("example.com", 1)}),
		route53fake.WithRecords(zoneIdOf("example.com"), []types.ResourceRecordSet{
			simpleRecord("example.com.", types.RRTypeA, "192.0.2.1"),
		}),
	)
	var view = newHostedZonesTest(t, api)
	var app = view.HostedZonesTable.serviceCtx.App
	loadZones(t, view, 1)
	awaitDetailsLoaded(t, app, view.DetailsTable.DetailsTable)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.RecordsTable.ErrorMessageCallback = callback })
	api.SetErr(errors.New("Throttling: route53:ListResourceRecordSets"))

	openRecordsTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "route53:ListResourceRecordSets") {
		t.Errorf("error = %q, want the error of the api", got)
	}

	api.SetErr(nil)
	for range 2 {
		uitest.PressKey(t, app, tcell.KeyTab)
	}
	uitest.AwaitRows(t, app, view.RecordsTable, []core.TableRow{
		recordHeadings, {"example.com", "A", "Simple", "", "300", "192.0.2.1"},
	})
}

func TestHostedZonesTable__OpenLoadsTheZonesToSelectOneNotLoaded(t *testing.T) {
	var api = route53fake.New(route53fake.WithHostedZones([]types.HostedZone{
		newZone("example.com", 1), newZone("example.net", 1),
	}))
	var view = newHostedZonesTest(t, api)
	var app = view.HostedZonesTable.serviceCtx.App

	// An ARN has the id without "/hostedzone/"
	uitest.OnUI(t, app, func() { view.HostedZonesTable.Open("ZEXAMPLENET") })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.HostedZonesTable.GetSelectedZone().Name) == "example.net."
	}, "example.net to be selected")
}

func TestHostedZonesTable__OpenOfAMissingZoneIsAnError(t *testing.T) {
	var api = route53fake.New(route53fake.WithHostedZones([]types.HostedZone{newZone("example.com", 1)}))
	var view = newHostedZonesTest(t, api)
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, view.HostedZonesTable.serviceCtx.App, func() {
		view.HostedZonesTable.ErrorMessageCallback = callback
		view.HostedZonesTable.Open("ZGONE")
	})

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "ZGONE") {
		t.Errorf("error = %q, want it to name the hosted zone", got)
	}
}

func newHealthChecksTest(t *testing.T, api *route53fake.Api) *HealthChecksPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.Route53Api](api)
	var view = NewHealthChecksPageView(
		NewHealthChecksTable(serviceCtx),
		NewHealthCheckDetailsTable(serviceCtx),
		NewHealthCheckStatusTable(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	return view
}

// loadHealthChecks loads the health checks of the page and waits for them
func loadHealthChecks(t *testing.T, view *HealthChecksPageView, count int) {
	t.Helper()
	var app = view.HealthChecksTable.serviceCtx.App
	view.HealthChecksTable.RefreshHealthChecks()
	uitest.Eventually(t, app, func() bool {
		return view.HealthChecksTable.GetTable().GetRowCount() == count+1
	}, "the health checks to be loaded")
}

func openStatusTab(t *testing.T, view *HealthChecksPageView) {
	t.Helper()
	switchTab(t, view.HealthChecksTable.serviceCtx.App, view.DetailsTable, view.TabView, HealthCheckTabNameStatus)
}

func TestHealthChecksTable__ResetLoadsTheHealthChecks(t *testing.T) {
	var api = route53fake.New(route53fake.WithHealthChecks([]types.HealthCheck{
		httpsCheck("hc-api", "api.example.com"),
		calculatedCheck("hc-all", 1, "hc-api", "hc-shop"),
	}))
	var view = newHealthChecksTest(t, api)
	var app = view.HealthChecksTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.HealthChecksTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.HealthChecksTable, []core.TableRow{
		healthCheckHeadings,
		{"hc-api", "HTTPS", "api.example.com:443/health"},
		{"hc-all", "CALCULATED", "1 of 2 child health checks"},
	})
}

func TestHealthChecksPage__SelectingAnEndpointCheckShowsItsEndpoint(t *testing.T) {
	var api = route53fake.New(route53fake.WithHealthChecks([]types.HealthCheck{httpsCheck("hc-api", "api.example.com")}))
	var view = newHealthChecksTest(t, api)
	var app = view.HealthChecksTable.serviceCtx.App

	view.HealthChecksTable.RefreshHealthChecks()

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Id", "hc-api"},
		{"ARN", "arn:aws:route53:::healthcheck/hc-api"},
		{"Type", "HTTPS"},
		{"Disabled", "false"},
		{"Inverted", "false"},
		{"Version", "3"},
		{"Linked Service", ""},
		{"Endpoint"},
		{"Target", "api.example.com:443/health"},
		{"IP Address", ""},
		{"Domain Name", "api.example.com"},
		{"Port", "443"},
		{"Resource Path", "/health"},
		{"Search String", ""},
		{"Enable SNI", "true"},
		{"Request Interval (s)", "30"},
		{"Failure Threshold", "3"},
		{"Measure Latency", "false"},
		{"Checker Regions", "eu-west-1, us-east-1"},
	})
}

func TestHealthChecksPage__CalculatedCheckLinksToItsChildren(t *testing.T) {
	var api = route53fake.New(route53fake.WithHealthChecks([]types.HealthCheck{
		calculatedCheck("hc-all", 1, "hc-api", "hc-shop"),
	}))
	var view = newHealthChecksTest(t, api)
	var app = view.HealthChecksTable.serviceCtx.App

	view.HealthChecksTable.RefreshHealthChecks()

	var details = view.DetailsTable.DetailsTable
	uitest.AwaitDetailsRows(t, app, details, []core.TableRow{
		{"Id", "hc-all"},
		{"ARN", "arn:aws:route53:::healthcheck/hc-all"},
		{"Type", "CALCULATED"},
		{"Disabled", "false"},
		{"Inverted", "false"},
		{"Version", "1"},
		{"Linked Service", ""},
		{"Child Health Checks"},
		{"Healthy When", "1 or more are healthy"},
		{"Child", "hc-api"},
		{"Child", "hc-shop"},
	})
	uitest.OnUI(t, app, func() {
		for row, want := range map[int]string{
			9:  "arn:aws:route53:::healthcheck/hc-api",
			10: "arn:aws:route53:::healthcheck/hc-shop",
		} {
			if link, _ := details.GetLink(row); link != want {
				t.Errorf("link of row %d = %q, want %q", row, link, want)
			}
		}
	})
}

func TestHealthChecksPage__StatusIsLoadedEachTimeItsTabIsOpened(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = route53fake.New(
			route53fake.WithHealthChecks([]types.HealthCheck{httpsCheck("hc-api", "api.example.com")}),
			route53fake.WithStatus("hc-api", []types.HealthCheckObservation{
				observation(types.HealthCheckRegionEuWest1, "15.177.62.1", "Success: HTTP Status Code 200, OK"),
			}),
		)
		var view = newHealthChecksTest(t, api)
		var app = view.HealthChecksTable.serviceCtx.App
		loadHealthChecks(t, view, 1)
		uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
			return len(api.StatusesAskedFor()) == 0
		}, "the status not to be loaded before the tab is opened")

		openStatusTab(t, view)

		var checked = core.FormatTime(aws.Time(time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)))
		uitest.AwaitRows(t, app, view.StatusTable, []core.TableRow{
			statusHeadings,
			{"eu-west-1", "15.177.62.1", checked, "Success: HTTP Status Code 200, OK"},
		})

		// The status has changed by the time the tab is opened again
		api.SetStatus("hc-api", []types.HealthCheckObservation{
			observation(types.HealthCheckRegionEuWest1, "15.177.62.1", "Failure: Connection timed out"),
		})
		uitest.PressKey(t, app, tcell.KeyTab)
		uitest.PressKey(t, app, tcell.KeyTab)

		// AwaitRows only waits for the number of rows, which is the same as before
		uitest.Eventually(t, app, func() bool {
			return view.StatusTable.GetCellText(1, 3) == "Failure: Connection timed out"
		}, "the new status to be shown")
		uitest.OnUI(t, app, func() {
			uitest.AssertTableRows(t, view.StatusTable, []core.TableRow{
				statusHeadings,
				{"eu-west-1", "15.177.62.1", checked, "Failure: Connection timed out"},
			})
		})
	})
}

func TestHealthChecksPage__CalculatedCheckHasNoCheckersToAsk(t *testing.T) {
	uitest.RunInBubble(t, func(t *testing.T) {
		var api = route53fake.New(route53fake.WithHealthChecks([]types.HealthCheck{
			calculatedCheck("hc-all", 1, "hc-api"),
			httpsCheck("hc-api", "api.example.com"),
		}))
		var view = newHealthChecksTest(t, api)
		var app = view.HealthChecksTable.serviceCtx.App
		loadHealthChecks(t, view, 2)
		openStatusTab(t, view)

		// Moving to a check with checkers shows that its status is asked for, and
		// that the calculated one before it wasn't
		uitest.OnUI(t, app, func() { view.HealthChecksTable.Select(2, 0) })

		uitest.Eventually(t, app, func() bool { return slices.Contains(api.StatusesAskedFor(), "hc-api") },
			"the status of hc-api to be loaded")
		if asked := api.StatusesAskedFor(); slices.Contains(asked, "hc-all") {
			t.Errorf("status asked for = %v, want not the calculated check", asked)
		}
	})
}

func TestIsOpenableRoute53Resource(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     bool
	}{
		{"arn:aws:route53:::hostedzone/Z0123456789ABC", true},
		{"arn:aws:route53:::healthcheck/abcd-1234", true},
		{"arn:aws:route53:::trafficpolicy/abcd", false},
		{"arn:aws:route53:::hostedzone/", false},
	} {
		var resource, err = arn.Parse(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		if got := isOpenableRoute53Resource(resource); got != test.want {
			t.Errorf("isOpenableRoute53Resource(%q) = %t, want %t", test.resource, got, test.want)
		}
	}
}

func TestHomeView__ResetLoadsTheHostedZones(t *testing.T) {
	var api = route53fake.New(route53fake.WithHostedZones([]types.HostedZone{newZone("storefront.example", 5)}))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewRoute53HomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "Zone of storefront.example")
}

func TestHomeView__OpeningAHealthCheckShowsItOnTheHealthChecksPage(t *testing.T) {
	var api = route53fake.New(route53fake.WithHealthChecks([]types.HealthCheck{
		httpsCheck("hc-api", "api.example.com"),
		httpsCheck("hc-shop", "shop.example.com"),
	}))
	var appCtx = uitest.NewAppContext()
	var page = NewRoute53HomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	var opener, ok = page.(core.ResourceOpener)
	var resource, _ = arn.Parse("arn:aws:route53:::healthcheck/hc-shop")
	if !ok || !opener.OpensArn(resource) {
		t.Fatal("Route 53 page should open health checks")
	}

	uitest.OnUI(t, appCtx.App, func() { opener.OpenResource(resource) })

	// The details of hc-shop, which only it has, are shown
	uitest.AwaitScreenText(t, appCtx.App, screen, "shop.example.com:443/health")
}
