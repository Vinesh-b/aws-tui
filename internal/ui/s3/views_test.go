package s3

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestHomeView__ResetKeyLoadsBucketsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewS3bucketsHomeView(appCtx, &fakeS3BucketsApi{buckets: []types.Bucket{newBucket("orders-bucket")}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders-bucket")
}
