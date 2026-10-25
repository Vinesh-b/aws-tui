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

	view.ApplyTheme(appContext.Theme)

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

// ApplyTheme colours the list from the theme: the text, the second line and the
// shortcut of each item, and the selected row, which is lit by the background
// that is used for the selected row of the tables. It replaces the colours the
// list was given.
func (inst *FilterableList) ApplyTheme(theme *AppTheme) {
	applyListTheme(inst.list, theme)
	inst.list.
		SetSecondaryTextStyle(textStyle(theme.PlaceholderTextColour, theme)).
		SetSelectedTextColor(theme.SecondaryTextColour).
		SetSelectedBackgroundColor(theme.MoreContrastBackgroundColor)
}

// textStyle is a style for text of the given colour on the background of the
// theme. A style carries the background it was made with, so a colour that is
// changed on its own leaves the background of the theme the view was built with.
func textStyle(colour tcell.Color, theme *AppTheme) tcell.Style {
	return tcell.StyleDefault.Foreground(colour).Background(theme.BackgroundColour)
}

// applyListTheme sets the text and the shortcuts of a list, whose styles each
// hold a background that is not changed by the one of the list.
func applyListTheme(list *tview.List, theme *AppTheme) {
	list.
		SetMainTextStyle(textStyle(theme.PrimaryTextColour, theme)).
		SetShortcutStyle(textStyle(theme.SecondaryTextColour, theme))
}

// SetCurrentItem moves the cursor to the item at the index.
func (inst *FilterableList) SetCurrentItem(index int) {
	inst.list.SetCurrentItem(index)
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
