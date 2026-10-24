package services

import (
	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewServicesHomeView(appContext *core.AppContext) *core.FilterableList {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedTextColor(appContext.Theme.TertiaryTextColour).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorGrey)

	servicesList.SetBorderPadding(0, 0, 1, 1)

	return core.NewFilterableList(servicesList, appContext)
}
