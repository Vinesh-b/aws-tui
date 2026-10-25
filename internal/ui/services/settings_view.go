package services

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const currentThemeMarker = "● in use"

// NewSettingsView lets the user change the settings of the app. Picking a theme
// applies it straight away.
func NewSettingsView(appCtx *core.AppContext) core.ServicePage {
	appCtx.Theme.ChangeColourScheme(tcell.NewHexColor(0x7A7A7A))
	defer appCtx.Theme.ResetGlobalStyle()

	var listView = tview.NewList().
		SetHighlightFullLine(true)

	listView.
		SetBorder(true).
		SetBorderPadding(1, 0, 1, 1).
		SetTitle("Theme")

	var themeList = core.NewFilterableList(listView, appCtx)
	var serviceView = core.NewServicePageView(appCtx)

	var currentIdx = 0
	for i, preset := range core.ThemePresets() {
		var marker = ""
		if preset.Name == appCtx.Theme.Name {
			marker = currentThemeMarker
			currentIdx = i
		}

		themeList.AddItem(preset.Name, marker, 0, func() {
			appCtx.SetTheme(preset)
		})
	}
	themeList.SetCurrentItem(currentIdx)

	serviceView.MainPage.AddItem(themeList, 0, 1, true)
	serviceView.InitViewNavigation(
		[][]core.View{
			{themeList},
		},
	)

	var serviceRootView = core.NewServiceRootView(string(core.SETTINGS), appCtx)
	serviceRootView.
		AddAndSwitchToPage("Theme", serviceView, true)
	serviceRootView.InitPageNavigation()

	return serviceRootView
}
