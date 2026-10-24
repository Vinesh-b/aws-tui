package ssm

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestHomeView__ResetKeyLoadsParametersFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewSystemManagerHomeView(appCtx, &fakeSystemsManagerApi{parameters: []types.Parameter{newParameter("/orders/db", 1)}})
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "/orders/db")
}
