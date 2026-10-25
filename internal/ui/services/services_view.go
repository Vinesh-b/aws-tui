package services

import (
	"aws-tui/internal/ui/core"

	"github.com/rivo/tview"
)

func NewServicesHomeView(appContext *core.AppContext) *core.FilterableList {
	var servicesList = tview.NewList().
		SetHighlightFullLine(true)

	servicesList.SetBorderPadding(0, 0, 1, 1)

	return core.NewFilterableList(servicesList, appContext)
}
