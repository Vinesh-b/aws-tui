package vpc

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestHomeView__ResetKeyLoadsVpcsFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewVpcHomeView(appCtx, &fakeEc2Api{vpcs: []types.Vpc{newVpc("vpc-orders", nil)}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "vpc-orders")
}
