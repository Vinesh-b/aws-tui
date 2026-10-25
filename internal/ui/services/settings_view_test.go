package services

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func keepGlobalStyle(t *testing.T) {
	t.Helper()
	var styles, borders = tview.Styles, tview.Borders
	t.Cleanup(func() { tview.Styles, tview.Borders = styles, borders })
}

func newSettingsAppContext() *core.AppContext {
	var appCtx = uitest.NewAppContext()
	*appCtx.Theme = core.DefaultTheme()
	return appCtx
}

func TestSettings__ListsThemesAndMarksTheOneInUse(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	var screen = uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx))
	appCtx.App.QueueUpdateDraw(func() {})

	for _, preset := range core.ThemePresets() {
		uitest.AwaitScreenText(t, appCtx.App, screen, preset.Name)
	}
	uitest.AwaitScreenText(t, appCtx.App, screen, currentThemeMarker)
}

func TestSettings__SelectingAThemeAppliesIt(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	var changes = 0
	appCtx.OnThemeChanged(func() { changes++ })
	uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx))

	uitest.PressRune(t, appCtx.App, 'j')
	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.Eventually(t, appCtx.App, func() bool { return appCtx.Theme.Name == "High Contrast" })
	if changes != 1 {
		t.Errorf("theme listeners ran %d times, want 1", changes)
	}
}

func TestSettings__StartsOnTheThemeInUse(t *testing.T) {
	keepGlobalStyle(t)
	var appCtx = newSettingsAppContext()
	appCtx.SetTheme(core.ThemePresets()[2])
	uitest.ShowPage(t, appCtx.App, NewSettingsView(appCtx))

	uitest.PressKey(t, appCtx.App, tcell.KeyEnter)

	uitest.Eventually(t, appCtx.App, func() bool { return appCtx.Theme.Name == "Nord" })
}
