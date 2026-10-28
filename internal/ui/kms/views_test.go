package kms

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
)

func TestHomeView__ResetKeyLoadsKeysFromApi(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewKmsHomeView(appCtx, newApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)

	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)

	uitest.AwaitScreenText(t, appCtx.App, screen, "alias/orders")
}

func TestHomeView__EnterShowsThePolicyOfTheSelectedKey(t *testing.T) {
	var appCtx = uitest.NewAppContext()
	var page = NewKmsHomeView(appCtx, newApi())
	var screen = uitest.ShowPage(t, appCtx.App, page)
	uitest.PressRune(t, appCtx.App, core.APP_KEY_BINDINGS.Reset)
	uitest.AwaitScreenText(t, appCtx.App, screen, "alias/orders")

	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.AwaitScreenText(t, appCtx.App, screen, "2012-10-17")
}
