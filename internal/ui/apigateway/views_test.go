package apigateway

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func TestHomeView__ResetKeyLoadsApisFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewApiGatewayHomeView(appCtx, newFake())
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "orders")
}
