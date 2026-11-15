package s3

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/s3/s3fake"
	"aws-tui/internal/ui/uitest"

	s3controlTypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/gdamore/tcell/v2"
)

const reportsPolicy = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123456789012:role/reader"},"Action":"s3:GetObject","Resource":"arn:aws:s3:us-east-1:123456789012:accesspoint/reports/object/*"}]}`

// newAccessPointsPageTest shows the page with its access points loaded, so its
// tabs can be opened with the keys
func newAccessPointsPageTest(t *testing.T, api *s3fake.Api) *AccessPointsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.S3BucketsApi](api)
	var view = NewAccessPointsPageView(
		NewAccessPointsTable(serviceCtx),
		NewAccessPointDetailsTable(serviceCtx),
		NewAccessPointPolicyView(serviceCtx),
		serviceCtx,
	)
	uitest.ShowPage(t, serviceCtx.App, view)
	view.AccessPointsTable.RefreshAccessPoints()
	uitest.Eventually(t, serviceCtx.App, func() bool {
		return view.AccessPointsTable.GetTable().GetRowCount() > 1
	}, "the access points to be loaded")
	return view
}

// openPolicyTab moves from the details tab to the policy tab
func openPolicyTab(t *testing.T, view *AccessPointsPageView) {
	t.Helper()
	var app = view.AccessPointsTable.serviceCtx.App
	uitest.OnUI(t, app, func() { app.SetFocus(view.DetailsTable) })
	uitest.PressKey(t, app, tcell.KeyTab)
}

func awaitPolicyText(t *testing.T, view *AccessPointsPageView, want string) {
	t.Helper()
	uitest.Eventually(t, view.AccessPointsTable.serviceCtx.App, func() bool {
		return strings.Contains(view.PolicyView.GetText(), want)
	}, "the policy view to show "+want)
}

func TestAccessPointsPage__PolicyIsOnlyLoadedOnceItsTabIsOpened(t *testing.T) {
	var api = s3fake.New(
		s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{newAccessPoint("reports", "orders")}),
		s3fake.WithAccessPointPolicy(accessPointArnOf("reports"), reportsPolicy),
	)
	var view = newAccessPointsPageTest(t, api)
	uitest.Eventually(t, view.AccessPointsTable.serviceCtx.App, func() bool {
		return slices.Contains(api.AccessPointsAskedFor(), accessPointArnOf("reports"))
	}, "the details of reports to be loaded")
	if asked := api.PoliciesAskedFor(); len(asked) != 0 {
		t.Fatalf("policies loaded before the tab was opened: %v", asked)
	}
	awaitPolicyText(t, view, "The policy loads when this tab is opened")

	openPolicyTab(t, view)

	// The policy is laid out over lines
	awaitPolicyText(t, view, `"Action": "s3:GetObject"`)
	if !view.PolicyView.JsonHighlighting() {
		t.Error("the policy is not highlighted as json")
	}
	uitest.PressKey(t, view.AccessPointsTable.serviceCtx.App, tcell.KeyTab)
	uitest.PressKey(t, view.AccessPointsTable.serviceCtx.App, tcell.KeyTab)
	if asked := api.PoliciesAskedFor(); len(asked) != 1 {
		t.Errorf("policies loaded after reopening the tab = %v, want reports once", asked)
	}
}

func TestAccessPointsPage__AccessPointWithoutAPolicySaysSo(t *testing.T) {
	var api = s3fake.New(s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{newAccessPoint("open-data", "orders")}))
	var view = newAccessPointsPageTest(t, api)

	openPolicyTab(t, view)

	awaitPolicyText(t, view, "This access point has no policy")
}

func TestAccessPointsPage__SelectingAnotherAccessPointWithThePolicyOpenLoadsItsPolicy(t *testing.T) {
	var api = s3fake.New(
		s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{
			newAccessPoint("analytics", "orders"),
			newAccessPoint("reports", "orders"),
		}),
		s3fake.WithAccessPointPolicy(accessPointArnOf("analytics"), `{"Statement":"analytics-only"}`),
		s3fake.WithAccessPointPolicy(accessPointArnOf("reports"), reportsPolicy),
	)
	var view = newAccessPointsPageTest(t, api)
	openPolicyTab(t, view)
	awaitPolicyText(t, view, "analytics-only")

	uitest.OnUI(t, view.AccessPointsTable.serviceCtx.App, func() { view.AccessPointsTable.Select(2, 0) })

	awaitPolicyText(t, view, `"Action": "s3:GetObject"`)
}

func TestAccessPointsPage__MovingThroughAccessPointsWithThePolicyOpenLoadsOnlyTheOneItStopsOn(t *testing.T) {
	var api = s3fake.New(s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{
		newAccessPoint("analytics", "orders"),
		newAccessPoint("billing", "orders"),
		newAccessPoint("network", "orders"),
		newAccessPoint("security", "orders"),
	}))
	var view = newAccessPointsPageTest(t, api)
	var app = view.AccessPointsTable.serviceCtx.App
	openPolicyTab(t, view)
	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.PoliciesAskedFor(), accessPointArnOf("analytics"))
	}, "the policy of analytics to be loaded")
	var before = len(api.PoliciesAskedFor())

	uitest.OnUI(t, app, func() {
		for row := 2; row <= 4; row++ {
			view.AccessPointsTable.Select(row, 0)
		}
	})

	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.PoliciesAskedFor()[before:], accessPointArnOf("security"))
	}, "the policy of security to be loaded")
	uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
		var asked = api.PoliciesAskedFor()[before:]
		return !slices.Contains(asked, accessPointArnOf("network")) && asked[len(asked)-1] == accessPointArnOf("security")
	}, "only the first and last access points of the move to have their policy loaded")
}

func TestAccessPointsPage__PolicyErrorIsShownAndLoadedAgainWhenTheTabIsReopened(t *testing.T) {
	var api = s3fake.New(
		s3fake.WithAccessPoints([]s3controlTypes.AccessPoint{newAccessPoint("reports", "orders")}),
		s3fake.WithAccessPointPolicy(accessPointArnOf("reports"), reportsPolicy),
	)
	var view = newAccessPointsPageTest(t, api)
	var app = view.AccessPointsTable.serviceCtx.App
	var callback, received = uitest.CaptureErrors()
	uitest.OnUI(t, app, func() { view.PolicyView.ErrorMessageCallback = callback })
	api.SetErr(errors.New("AccessDenied: s3:GetAccessPointPolicy"))

	openPolicyTab(t, view)

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "s3:GetAccessPointPolicy") {
		t.Errorf("error = %q, want the error of the api", got)
	}
	awaitPolicyText(t, view, "Failed to load the policy")

	api.SetErr(nil)
	uitest.PressKey(t, app, tcell.KeyTab)
	uitest.PressKey(t, app, tcell.KeyTab)
	awaitPolicyText(t, view, `"Action": "s3:GetObject"`)
}
