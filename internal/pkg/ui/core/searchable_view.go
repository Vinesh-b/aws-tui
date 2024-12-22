package core

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const SEARCH_PAGE_NAME = "SEARCH"

type SearchableView struct {
	RootView *tview.Flex
	MainPage *tview.Flex

	searchInput *tview.InputField
	showSearch  bool
	pages       *tview.Pages
	app         *tview.Application
	Logger      *log.Logger
}

func NewSearchableView(
	app *tview.Application,
	logger *log.Logger,
	mainPage *tview.Flex,
) *SearchableView {
	var floatingSearch = NewFloatingSearchView("Search", 70, 3)
	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(SEARCH_PAGE_NAME, floatingSearch.RootView, true, false)

	var view = &SearchableView{
		RootView: tview.NewFlex().AddItem(pages, 0, 1, true),
		MainPage: mainPage,

		searchInput: floatingSearch.InputField,
		showSearch:  true,
		pages:       pages,
		app:         app,
		Logger:      logger,
	}

	view.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			if view.showSearch {
				pages.ShowPage(SEARCH_PAGE_NAME)
			} else {
				pages.HidePage(SEARCH_PAGE_NAME)
			}
			view.showSearch = !view.showSearch
			return nil
		}
		return event
	})

	return view
}

func (inst *SearchableView) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return inst.searchInput.SetInputCapture(capture)
}

func (inst *SearchableView) SetDoneFunc(handler func(key tcell.Key)) *tview.InputField {
	var default_func = func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if !inst.showSearch {
				inst.pages.HidePage(SEARCH_PAGE_NAME)
				inst.showSearch = !inst.showSearch
			}
		}
		return
	}

	return inst.searchInput.SetDoneFunc(func(key tcell.Key) {
		default_func(key)
		handler(key)
	})
}

func (inst *SearchableView) GetText() string {
	return inst.searchInput.GetText()
}

func (inst *SearchableView) SetText(text string) *tview.InputField {
	return inst.searchInput.SetText(text)
}
