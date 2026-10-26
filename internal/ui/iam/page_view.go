package iam

import (
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

type errorCallback = func(text string, a ...any)

// newIamPageView puts the tabs of the selected resource above the list of the
// resources, and shows what fails in any of them as an error.
func newIamPageView(
	appCtx *core.AppContext,
	tabView *core.TabViewHorizontal,
	listTable core.View,
	errorCallbacks ...*errorCallback,
) *core.ServicePageView {
	const tabViewSize = 5000
	const listTableSize = 5000

	var mainPage = core.NewResizableView(
		tabView, tabViewSize,
		listTable, listTableSize,
		tview.FlexRow,
	)
	var serviceView = core.NewServicePageView(appCtx)
	serviceView.MainPage.AddItem(mainPage, 0, 1, true)

	var errorHandler = func(text string, a ...any) {
		serviceView.DisplayMessage(core.ErrorPrompt, text, a...)
	}
	for _, callback := range errorCallbacks {
		*callback = errorHandler
	}

	serviceView.InitViewNavigation(
		[][]core.View{
			{tabView},
			{listTable},
		},
	)

	return serviceView
}
