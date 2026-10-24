package core

import (
	"aws-tui/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilterableListItem struct {
	MainText      string
	SecondaryText string
	Shortcut      rune
	SelectedFunc  func()
}

type FilterableList struct {
	*SearchableView
	list  *tview.List
	items []FilterableListItem
}

func NewFilterableList(list *tview.List, appContext *AppContext) *FilterableList {
	var view = &FilterableList{
		SearchableView: NewSearchableView(list, appContext),
		list:           list,
		items:          []FilterableListItem{},
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var numItems = list.GetItemCount()
		if numItems == 0 || event.Key() != tcell.KeyRune {
			return event
		}

		var currentIdx = list.GetCurrentItem()
		switch event.Rune() {
		case APP_KEY_BINDINGS.MoveUpRune:
			list.SetCurrentItem((currentIdx - 1 + numItems) % numItems)
			return nil
		case APP_KEY_BINDINGS.MoveDownRune:
			list.SetCurrentItem((currentIdx + 1) % numItems)
			return nil
		}
		return event
	})

	view.SetSearchChangedFunc(view.showMatching)

	return view
}

func (inst *FilterableList) AddItem(
	mainText string, secondaryText string, shortcut rune, selected func(),
) {
	inst.list.AddItem(mainText, secondaryText, shortcut, selected)
	inst.items = append(inst.items, FilterableListItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		SelectedFunc:  selected,
	})
}

func (inst *FilterableList) IsEscapable() bool {
	return !inst.SearchableView.IsEscapable()
}

func (inst *FilterableList) showMatching(search string) {
	inst.list.Clear()

	var matches = inst.items
	if len(search) > 0 {
		matches = utils.FuzzySearch(search, inst.items, func(item FilterableListItem) string {
			return item.MainText + " " + item.SecondaryText
		})
	}

	for _, item := range matches {
		inst.list.AddItem(item.MainText, item.SecondaryText, item.Shortcut, item.SelectedFunc)
	}
}
