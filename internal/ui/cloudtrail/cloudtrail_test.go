package cloudtrail

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/cloudtrail/cloudtrailfake"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/gdamore/tcell/v2"
)

var eventHeadings = core.TableRow{"Time", "Event Name", "User", "Event Source", "Resource"}

const roleArn = "arn:aws:iam::123456789012:role/deployer"

func newEvent(id string, name string, minute int) types.Event {
	var at = time.Date(2026, 11, 14, 9, minute, 0, 0, time.UTC)
	return types.Event{
		EventId:     aws.String(id),
		EventName:   aws.String(name),
		EventTime:   &at,
		EventSource: aws.String("s3.amazonaws.com"),
		Username:    aws.String("deployer"),
		AccessKeyId: aws.String("ASIAEXAMPLE"),
		ReadOnly:    aws.String("false"),
		Resources: []types.Resource{
			{ResourceType: aws.String("AWS::S3::Bucket"), ResourceName: aws.String("orders-" + id)},
		},
		CloudTrailEvent: aws.String(`{
			"awsRegion": "us-east-1",
			"sourceIPAddress": "203.0.113.7",
			"userAgent": "aws-cli/2.17",
			"errorCode": "AccessDenied",
			"errorMessage": "Not allowed",
			"userIdentity": {"type": "AssumedRole", "arn": "` + roleArn + `", "accountId": "123456789012"},
			"requestParameters": {"bucketName": "orders-` + id + `"}
		}`),
	}
}

func trailArnOf(name string) string {
	return "arn:aws:cloudtrail:us-east-1:123456789012:trail/" + name
}

func newTrail(name string) types.Trail {
	return types.Trail{
		Name:                       aws.String(name),
		TrailARN:                   aws.String(trailArnOf(name)),
		HomeRegion:                 aws.String("us-east-1"),
		IsMultiRegionTrail:         aws.Bool(true),
		IsOrganizationTrail:        aws.Bool(false),
		IncludeGlobalServiceEvents: aws.Bool(true),
		LogFileValidationEnabled:   aws.Bool(true),
		S3BucketName:               aws.String(name + "-logs"),
	}
}

func TestEventRecord__TextFollowsThePathOfKeys(t *testing.T) {
	var record = parseEventRecord(`{"userIdentity":{"arn":"a","sessionContext":{"mfa":true}},"count":3,"list":[1]}`)

	for _, test := range []struct {
		path []string
		want string
	}{
		{[]string{"userIdentity", "arn"}, "a"},
		{[]string{"userIdentity", "sessionContext", "mfa"}, "true"},
		{[]string{"count"}, "3"},
		{[]string{"userIdentity"}, ""},
		{[]string{"list"}, ""},
		{[]string{"missing", "arn"}, ""},
		{[]string{"count", "deeper"}, ""},
	} {
		if got := record.text(test.path...); got != test.want {
			t.Errorf("text(%v) = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestEventRecord__TextOfARecordThatIsNotJsonIsEmpty(t *testing.T) {
	var record = parseEventRecord("not json")

	if record != nil || record.text("awsRegion") != "" {
		t.Errorf("record = %v, want none", record)
	}
}

func TestEventsTable__ResetLoadsTheNewestEvents(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithEventPages(
		[]types.Event{newEvent("e1", "PutBucketPolicy", 30), newEvent("e2", "CreateBucket", 20)},
	))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudTrailApi](api)
	var table = NewEventsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	uitest.OnUI(t, serviceCtx.App, func() { uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		eventHeadings,
		{"2026-11-14 09:30:00", "PutBucketPolicy", "deployer", "s3.amazonaws.com", "orders-e1"},
		{"2026-11-14 09:20:00", "CreateBucket", "deployer", "s3.amazonaws.com", "orders-e2"},
	})
}

func TestEventsPage__MovingPastTheLastEventLoadsTheNextPage(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithEventPages(
		[]types.Event{newEvent("e1", "PutBucketPolicy", 30), newEvent("e2", "CreateBucket", 20)},
		[]types.Event{newEvent("e3", "DeleteBucketReplication", 10)},
	))
	var appCtx = uitest.NewAppContext()
	var page = NewCloudTrailHomeView(appCtx, api)
	var screen = uitest.ShowPage(t, appCtx.App, page)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "CreateBucket")

	uitest.PressKey(t, appCtx.App, tcell.KeyDown)
	uitest.PressKey(t, appCtx.App, tcell.KeyDown)

	uitest.AwaitScreenText(t, appCtx.App, screen, "DeleteBucketReplication")
}

func TestEventsPage__SelectingAnEventShowsItsDetails(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithEventPages(
		[]types.Event{newEvent("e1", "PutBucketPolicy", 30), newEvent("e2", "CreateBucket", 20)},
	))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudTrailApi](api)
	var view = NewEventsPageView(NewEventsTable(serviceCtx), NewEventDetailsTable(serviceCtx.AppContext), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	view.EventsTable.RefreshEvents(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return view.EventsTable.GetTable().GetRowCount() == 3 })

	uitest.OnUI(t, serviceCtx.App, func() { view.EventsTable.Select(2, 0) })

	uitest.AwaitDetailsRows(t, serviceCtx.App, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Event Name", "CreateBucket"},
		{"Event Time", "2026-11-14 09:20:00"},
		{"Event Source", "s3.amazonaws.com"},
		{"User Name", "deployer"},
		{"Access Key", "ASIAEXAMPLE"},
		{"Read Only", "false"},
		{"Region", "us-east-1"},
		{"Source IP", "203.0.113.7"},
		{"User Agent", "aws-cli/2.17"},
		{"Error Code", "AccessDenied"},
		{"Error Message", "Not allowed"},
		{"Event ID", "e2"},
		{"Identity"},
		{"Type", "AssumedRole"},
		{"ARN", roleArn},
		{"Account", "123456789012"},
		{"Resources"},
		{"AWS::S3::Bucket", "orders-e2"},
	})
}

func TestEventsTable__ErrorIsShown(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithErr(errors.New("throttled")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudTrailApi](api)
	var table = NewEventsTable(serviceCtx)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshEvents(true)

	if got := uitest.AwaitError(t, received); got != "throttled" {
		t.Errorf("error = %q, want throttled", got)
	}
}

func newTrailsTest(t *testing.T, api *cloudtrailfake.Api) *TrailsPageView {
	t.Helper()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.CloudTrailApi](api)
	var view = NewTrailsPageView(NewTrailsTable(serviceCtx), NewTrailDetailsTable(serviceCtx), serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	return view
}

func TestTrailsTable__ResetLoadsTheTrails(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithTrails([]types.Trail{newTrail("audit"), newTrail("security")}))
	var view = newTrailsTest(t, api)
	var app = view.TrailsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { uitest.PressTableRune(view.TrailsTable, core.APP_KEY_BINDINGS.Reset) })

	uitest.AwaitRows(t, app, view.TrailsTable, []core.TableRow{
		{"Name", "Home Region", "Multi-Region", "S3 Bucket"},
		{"audit", "us-east-1", "true", "audit-logs"},
		{"security", "us-east-1", "true", "security-logs"},
	})
}

func TestTrailsPage__SelectingATrailShowsItThenItsStatus(t *testing.T) {
	var started = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var api = cloudtrailfake.New(
		cloudtrailfake.WithTrails([]types.Trail{newTrail("audit")}),
		cloudtrailfake.WithStatus(trailArnOf("audit"), &cloudtrail.GetTrailStatusOutput{
			IsLogging:           aws.Bool(true),
			StartLoggingTime:    &started,
			LatestDeliveryError: aws.String("AccessDenied"),
		}),
	)
	var view = newTrailsTest(t, api)
	var app = view.TrailsTable.serviceCtx.App

	view.TrailsTable.RefreshTrails(true)

	uitest.AwaitDetailsRows(t, app, view.DetailsTable.DetailsTable, []core.TableRow{
		{"Name", "audit"},
		{"Home Region", "us-east-1"},
		{"Multi-Region", "true"},
		{"Organization Trail", "false"},
		{"Global Service Events", "true"},
		{"Log File Validation", "true"},
		{"Custom Event Selectors", ""},
		{"Insight Selectors", ""},
		{"KMS Key", ""},
		{"ARN", trailArnOf("audit")},
		{"Delivery"},
		{"S3 Bucket", "audit-logs"},
		{"S3 Key Prefix", ""},
		{"SNS Topic", ""},
		{"Log Group", ""},
		{"Logs Role", ""},
		{"Status"},
		{"Logging", "true"},
		{"Started Logging", "2026-01-02 03:04:05"},
		{"Stopped Logging", ""},
		{"Latest Delivery", ""},
		{"Delivery Error", "AccessDenied"},
		{"Latest Digest", ""},
		{"Digest Error", ""},
		{"Latest Logs Delivery", ""},
		{"Logs Delivery Error", ""},
		{"Notification Error", ""},
	})
}

func TestTrailsPage__MovingThroughTrailsLoadsOnlyTheStatusOfTheTrailItStopsOn(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithTrails([]types.Trail{
		newTrail("audit"), newTrail("billing"), newTrail("network"), newTrail("security"),
	}))
	var view = newTrailsTest(t, api)
	var app = view.TrailsTable.serviceCtx.App
	view.TrailsTable.RefreshTrails(true)
	uitest.Eventually(t, app, func() bool { return view.TrailsTable.GetTable().GetRowCount() == 5 })
	var before = len(api.StatusesAskedFor())

	uitest.OnUI(t, app, func() {
		for row := 2; row <= 4; row++ {
			view.TrailsTable.Select(row, 0)
		}
	})
	// The trail is shown straight away, before its status
	uitest.OnUI(t, app, func() {
		if got := view.DetailsTable.GetCellText(0, 1); got != "security" {
			t.Errorf("trail shown = %q, want security", got)
		}
	})

	uitest.Eventually(t, app, func() bool {
		return slices.Contains(api.StatusesAskedFor()[before:], trailArnOf("security"))
	}, "the status of security to be loaded")
	uitest.Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
		var asked = api.StatusesAskedFor()[before:]
		return !slices.Contains(asked, trailArnOf("network")) && asked[len(asked)-1] == trailArnOf("security")
	}, "only the first and last trails of the move to have their status loaded")
}

func TestTrailsTable__OpenLoadsTheTrailsToSelectOneNotLoaded(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithTrails([]types.Trail{newTrail("audit"), newTrail("security")}))
	var view = newTrailsTest(t, api)
	var app = view.TrailsTable.serviceCtx.App

	uitest.OnUI(t, app, func() { view.TrailsTable.Open(trailArnOf("security")) })

	uitest.Eventually(t, app, func() bool {
		return aws.ToString(view.TrailsTable.GetSelectedTrail().Name) == "security"
	}, "security to be selected")
}

func TestTrailsTable__OpenOfAMissingTrailIsAnError(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithTrails([]types.Trail{newTrail("audit")}))
	var view = newTrailsTest(t, api)
	var callback, received = uitest.CaptureErrors()
	view.TrailsTable.ErrorMessageCallback = callback

	uitest.OnUI(t, view.TrailsTable.serviceCtx.App, func() { view.TrailsTable.Open(trailArnOf("gone")) })

	if got := uitest.AwaitError(t, received); !strings.Contains(got, "trail/gone") {
		t.Errorf("error = %q, want it to name the trail", got)
	}
}

func TestIsOpenableTrail(t *testing.T) {
	for _, test := range []struct {
		resource string
		want     bool
	}{
		{"trail/audit", true},
		{"trail/", false},
		{"eventdatastore/abc", false},
	} {
		var resource = arn.ARN{Service: "cloudtrail", Resource: test.resource}
		if got := isOpenableTrail(resource); got != test.want {
			t.Errorf("isOpenableTrail(%q) = %t, want %t", test.resource, got, test.want)
		}
	}
}

func TestHomeView__ResetLoadsTheEventHistory(t *testing.T) {
	var api = cloudtrailfake.New(cloudtrailfake.WithEventPages(
		[]types.Event{newEvent("e1", "PutBucketPolicy", 30)},
	))
	var appCtx = uitest.NewAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewCloudTrailHomeView(appCtx, api))

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "PutBucketPolicy")
}
