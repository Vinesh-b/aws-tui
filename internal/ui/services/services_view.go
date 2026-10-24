package services

import (
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServiceListItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	SelectedFunc  func()
}

type ServicesHomeView struct {
	*core.SearchableView
	filteredList     *tview.List
	serviceListItems []ServiceListItem
}

func NewServicesHomeView(appContext *core.AppContext) *ServicesHomeView {
	var servicesList = tview.NewList().
		SetSecondaryTextColor(tcell.ColorGrey).
		SetSelectedTextColor(appContext.Theme.TertiaryTextColour).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorGrey)

	servicesList.SetBorderPadding(0, 0, 1, 1)

	var view = &ServicesHomeView{
		SearchableView:   core.NewSearchableView(servicesList, appContext),
		filteredList:     servicesList,
		serviceListItems: []ServiceListItem{},
	}

	servicesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = servicesList.GetCurrentItem()
		var numItems = servicesList.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case core.APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			case core.APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				servicesList.SetCurrentItem(currentIdx)
				return nil
			}
		}
		return event
	})

	view.SetSearchChangedFunc(func(search string) {
		servicesList.Clear()

		if len(search) == 0 {
			for _, item := range view.serviceListItems {
				servicesList.AddItem(
					item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc,
				)
			}
			return
		}

		var filteredItems = utils.FuzzySearch(search, view.serviceListItems,
			func(listItem ServiceListItem) string {
				return listItem.MainText + " " + listItem.SecondaryText
			},
		)

		for _, item := range filteredItems {
			servicesList.AddItem(
				item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc,
			)
		}
		return
	})

	return view
}

func (inst *ServicesHomeView) AddItem(
	mainText string, secondaryText string, shortcut rune, selected func(),
) {
	inst.filteredList.AddItem(mainText, secondaryText, shortcut, selected)
	inst.serviceListItems = append(inst.serviceListItems, ServiceListItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		SelectedFunc:  selected,
	})
}

func (inst *ServicesHomeView) IsEscapable() bool {
	return !inst.SearchableView.IsEscapable()
}
