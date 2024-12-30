package core

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	SEARCH_PAGE_NAME = "SEARCH"
	MAIN_PAGE_NAME   = "MAIN_PAGE"
)

type SearchableView struct {
	*tview.Pages
	MainPage        tview.Primitive
	HighlightSearch bool

	searchInput     *tview.InputField
	showSearch      bool
	searchPositions []int
	app             *tview.Application
	logger          *log.Logger
}

func NewSearchableView(
	mainPage tview.Primitive,
) *SearchableView {
	var floatingSearch = NewFloatingSearchView("Search", 70, 3)
	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(SEARCH_PAGE_NAME, floatingSearch.RootView, true, false)

	var view = &SearchableView{
		Pages:           pages,
		MainPage:        mainPage,
		HighlightSearch: false,

		searchInput:     floatingSearch.InputField,
		showSearch:      true,
		searchPositions: nil,
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			if view.showSearch {
				view.ShowPage(SEARCH_PAGE_NAME)
			} else {
				view.HidePage(SEARCH_PAGE_NAME)
			}
			view.showSearch = !view.showSearch
			return nil
		}
		return event

	})

	view.SetSearchInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSearchDoneFunc(func(key tcell.Key) {})

	return view
}

func (inst *SearchableView) SetSearchInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.searchInput.SetInputCapture(capture)
}

func (inst *SearchableView) SetSearchDoneFunc(handler func(key tcell.Key)) {
	var default_func = func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if !inst.showSearch {
				inst.HidePage(SEARCH_PAGE_NAME)
				inst.showSearch = !inst.showSearch
			}
		}
		return
	}

	inst.searchInput.SetDoneFunc(func(key tcell.Key) {
		default_func(key)
		handler(key)
	})
}

func (inst *SearchableView) GetSearchText() string {
	return inst.searchInput.GetText()
}

func (inst *SearchableView) SetSearchText(text string) {
	inst.searchInput.SetText(text)
}
