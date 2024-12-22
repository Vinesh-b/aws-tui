package core

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchableView struct {
	RootView *tview.Flex
	MainPage *tview.Flex

	searchInput *tview.InputField
	pages       *tview.Pages
	app         *tview.Application
	Logger      *log.Logger
}

func NewSearchableView(
	app *tview.Application,
	logger *log.Logger,
	mainPage *tview.Flex,
) *SearchableView {
	var searchPageName = "SEARCH"
	var floatingSearch = NewFloatingSearchView("Search", 70, 3)
	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(searchPageName, floatingSearch.RootView, true, false)

	var showSearch = true

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			if showSearch {
				pages.ShowPage(searchPageName)
			} else {
				pages.HidePage(searchPageName)
			}
			showSearch = !showSearch
			return nil
		}
		return event
	})

	return &SearchableView{
		RootView: tview.NewFlex().AddItem(pages, 0, 1, true),
		MainPage: mainPage,

		searchInput: floatingSearch.InputField,
		pages:       pages,
		app:         app,
		Logger:      logger,
	}
}

func (inst *SearchableView) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return inst.searchInput.SetInputCapture(capture)
}

func (inst *SearchableView) SetDoneFunc(handler func(key tcell.Key)) *tview.InputField {
	return inst.searchInput.SetDoneFunc(handler)
}

func (inst *SearchableView) GetText() string {
	return inst.searchInput.GetText()
}

func (inst *SearchableView) SetText(text string) *tview.InputField {
	return inst.searchInput.SetText(text)
}
