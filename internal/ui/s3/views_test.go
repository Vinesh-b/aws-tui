package s3

import (
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/s3/s3fake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gdamore/tcell/v2"
)

func TestHomeView__ResetKeyLoadsBucketsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewS3bucketsHomeView(appCtx, s3fake.New(s3fake.WithBuckets([]types.Bucket{newBucket("orders-bucket")})))
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-bucket")
}

func TestHomeView__PreviewKeepsTheFocusWhenItsTabIsOpenedAndLoaded(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var api = s3fake.New(
		s3fake.WithBuckets([]types.Bucket{newBucket("orders-bucket")}),
		s3fake.WithObjects([]types.Object{newObject("a.txt", 5)}),
		s3fake.WithHead(newTextHead("text/plain", 5)),
		s3fake.WithContent([]byte("hello")),
	)
	var serviceCtx = core.NewServiceViewContext(appCtx, awsapi.S3BucketsApi(api))
	var preview = NewObjectPreviewView(serviceCtx)
	var details = NewS3bucketsDetailsView(
		NewBucketListTable(serviceCtx),
		NewBucketObjectsTable(serviceCtx),
		NewS3ObjectDetailsTable(serviceCtx),
		preview,
		serviceCtx,
	)
	var page = core.NewServiceRootView(string(core.S3BUCKETS), appCtx)
	page.AddAndSwitchToPage("S3Buckets", details, true)
	page.InitPageNavigation()
	details.InitInputCapture()
	var screen = uitest.ShowPage(t, appCtx.App, page)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-bucket")
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)
	uitest.AwaitScreenText(t, appCtx.App, screen, "a.txt")
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.MoveDownRune)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.MoveDownRune)
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)
	uitest.PressKey(t, appCtx.App, core.APP_KEY_BINDINGS.ViewFocusRight)

	uitest.PressKey(t, appCtx.App, tcell.KeyTab)

	uitest.AwaitScreenText(t, appCtx.App, screen, "hello")
	var focused bool
	uitest.OnUI(t, appCtx.App, func() { focused = preview.HasFocus() })
	if !focused {
		t.Error("the preview does not have the focus after its tab is opened and loaded")
	}
}
