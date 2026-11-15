package s3

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/s3/s3fake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controlTypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
)

func accessPointArnOf(name string) string {
	return "arn:aws:s3:us-east-1:123456789012:accesspoint/" + name
}

func newAccessPoint(name string, bucket string) s3controlTypes.AccessPoint {
	return s3controlTypes.AccessPoint{
		Name:            aws.String(name),
		Bucket:          aws.String(bucket),
		BucketAccountId: aws.String("123456789012"),
		NetworkOrigin:   s3controlTypes.NetworkOriginInternet,
		AccessPointArn:  aws.String(accessPointArnOf(name)),
		Alias:           aws.String(name + "-abc-s3alias"),
	}
}

func newVpcAccessPoint(name string, bucket string, vpc string) s3controlTypes.AccessPoint {
	var point = newAccessPoint(name, bucket)
	point.NetworkOrigin = s3controlTypes.NetworkOriginVpc
	point.VpcConfiguration = &s3controlTypes.VpcConfiguration{VpcId: aws.String(vpc)}
	return point
}

func newAccessPointsTest(t *testing.T, api *s3fake.Api) *AccessPointsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var view = NewAccessPointsPageView(
		NewAccessPointsTable(serviceCtx),
		NewAccessPointDetailsTable(serviceCtx),
		NewAccessPointPolicyView(serviceCtx),
		serviceCtx,
	)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestAccessPointsTable__ResetLoadsTheAccessPoints(t *testing.T) {
	var api = s3fake.New(s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{
		newAccessPoint("analytics", "orders"),
		newVpcAccessPoint("internal-reports", "reports", "vpc-0abc"),
	}))
	var view = newAccessPointsTest(t, api)
	var app = view.AccessPointsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.AccessPointsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.AccessPointsTable, []core.TableRow{
		{"Name", "Bucket", "Network Origin", "VPC"},
		{"analytics", "orders", "Internet", ""},
		{"internal-reports", "reports", "VPC", "vpc-0abc"},
	})
}

func TestAccessPointsPage__SelectingAnAccessPointShowsItThenItsDetails(t *testing.T) {
	var created = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var api = s3fake.New(
		s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{newVpcAccessPoint("internal-reports", "reports", "vpc-0abc")}),
		s3fake.WithAccessPointDetails(accessPointArnOf("internal-reports"), s3control.GetAccessPointOutput{
			CreationDate: &created,
			PublicAccessBlockConfiguration: &s3controlTypes.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(false),
				RestrictPublicBuckets: aws.Bool(true),
			},
			Endpoints: map[string]string{
				"ipv4":      "internal-reports-123456789012.s3-accesspoint.us-east-1.amazonaws.com",
				"dualstack": "internal-reports-123456789012.s3-accesspoint.dualstack.us-east-1.amazonaws.com",
			},
		}),
	)
	var view = newAccessPointsTest(t, api)
	var app = view.AccessPointsTable.serviceCtx.App

	view.AccessPointsTable.RefreshAccessPoints()

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "internal-reports"},
		{"Bucket", "reports"},
		{"Bucket Account", "123456789012"},
		{"Network Origin", "VPC"},
		{"VPC", "vpc-0abc"},
		{"Alias", "internal-reports-abc-s3alias"},
		{"ARN", accessPointArnOf("internal-reports")},
		{"Created", "2026-01-02 03:04:05"},
		{"Block Public Access"},
		{"Block Public ACLs", "true"},
		{"Ignore Public ACLs", "true"},
		{"Block Public Policy", "false"},
		{"Restrict Public Buckets", "true"},
		{"Endpoints"},
		{"dualstack", "internal-reports-123456789012.s3-accesspoint.dualstack.us-east-1.amazonaws.com"},
		{"ipv4", "internal-reports-123456789012.s3-accesspoint.us-east-1.amazonaws.com"},
	})
}

func TestAccessPointsPage__MovingThroughAccessPointsLoadsOnlyTheOneItStopsOn(t *testing.T) {
	var api = s3fake.New(s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{
		newAccessPoint("analytics", "orders"),
		newAccessPoint("billing", "orders"),
		newAccessPoint("network", "orders"),
		newAccessPoint("security", "orders"),
	}))
	var view = newAccessPointsTest(t, api)
	var app = view.AccessPointsTable.serviceCtx.App
	view.AccessPointsTable.RefreshAccessPoints()
	uitest.Eventually(t, app, func() bool { return view.AccessPointsTable.GetTable().GetRowCount() == 5 })
	var before = len(api.AccessPointsAskedFor())

	uitest.OnUI(t, app, func() {
		for row := 2; row <= 4; row++ {
			view.AccessPointsTable.Select(row, 0)
		}
	})
	// The access point is shown straight away, before its details
	uitest.OnUI(t, app, func() {
		if got := view.DetailsTable.GetCellText(0, 1); got != "security" {
			t.Errorf("access point shown = %q, want security", got)
		}
	})

	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.AccessPointsAskedFor()[before:], accessPointArnOf("security"))
	}, "the details of security to be loaded")
	uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
		var asked = api.AccessPointsAskedFor()[before:]
		return !slices.Contains(asked, accessPointArnOf("network")) && asked[len(asked)-1] == accessPointArnOf("security")
	}, "only the first and last access points of the move to have their details loaded")
}

func TestAccessPointsTable__ErrorIsShown(t *testing.T) {
	var api = s3fake.New(s3fake.WithErr(errors.New("AccessDenied: s3:ListAccessPoints")))
	var view = newAccessPointsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.AccessPointsTable.ErrorMessageCallback = callback

	view.AccessPointsTable.RefreshAccessPoints()

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "s3:ListAccessPoints") {
		t.Errorf("error = %q, want the error of the api", got)
	}
}

func TestHomeView__AccessPointsPageLoadsTheAccessPoints(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = s3fake.New(s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{newAccessPoint("analytics-ap", "orders")}))
	var page = NewS3bucketsHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)

	var paged, ok = page.(core.PagedServicePage)
	if !ok {
		t.Fatal("the S3 home view has no pages")
	}
	if pages := paged.ServicePages(); !slices.Contains(pages, "Access Points") {
		t.Fatalf("pages = %v, want an Access Points page", pages)
	}
	uitest.OnUI(t, appCtx.App, func() { paged.ShowServicePage("Access Points") })
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "analytics-ap")
}
