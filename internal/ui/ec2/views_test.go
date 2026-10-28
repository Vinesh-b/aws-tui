package ec2

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsInstancesFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewEc2HomeView(appCtx, newApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "i-2")
}
