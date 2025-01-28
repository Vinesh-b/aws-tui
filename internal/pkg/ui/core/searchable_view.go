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

	searchInput       *tview.InputField
	isSearchHidden    bool
	searchDoneHandler func(key tcell.Key)
	app               *tview.Application
	logger            *log.Logger
}

func NewSearchableView(
	mainPage tview.Primitive,
	app *tview.Application,
) *SearchableView {
	var floatingSearch = NewFloatingSearchView("Search", 0, 3)
	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(SEARCH_PAGE_NAME, floatingSearch.RootView, true, false)

	var view = &SearchableView{
		Pages:           pages,
		MainPage:        mainPage,
		HighlightSearch: false,

		searchInput:       floatingSearch.InputField,
		isSearchHidden:    true,
		searchDoneHandler: func(key tcell.Key) {},
		app:               app,
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Find:
			if view.isSearchHidden {
				view.ShowPage(SEARCH_PAGE_NAME)
				view.app.SetFocus(view.searchInput)
			} else {
				view.HidePage(SEARCH_PAGE_NAME)
			}
			view.isSearchHidden = !view.isSearchHidden
			return nil
		}
		return event

	})

	view.SetSearchInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSearchDoneFunc(func(key tcell.Key) {})

	return view
}

func (inst *SearchableView) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	var searchToggle = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Find:
			if inst.isSearchHidden {
				inst.ShowPage(SEARCH_PAGE_NAME)
				inst.app.SetFocus(inst.searchInput)
			} else {
				inst.HidePage(SEARCH_PAGE_NAME)
			}
			inst.isSearchHidden = !inst.isSearchHidden
			return nil
		}
		return event
	}

	inst.Pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event = searchToggle(event); event == nil {
			return nil
		}

		return capture(event)
	})
}

func (inst *SearchableView) SetSearchInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			inst.HidePage(SEARCH_PAGE_NAME)
			inst.isSearchHidden = true
			return nil
		}

		return capture(event)
	})
}

func (inst *SearchableView) SetSearchDoneFunc(handler func(key tcell.Key)) {
	var default_func = func(key tcell.Key) {
		switch key {
		case APP_KEY_BINDINGS.Done:
			if !inst.isSearchHidden {
				inst.HidePage(SEARCH_PAGE_NAME)
				inst.isSearchHidden = !inst.isSearchHidden
			}
		}
		return
	}

	inst.searchDoneHandler = handler

	inst.searchInput.SetDoneFunc(func(key tcell.Key) {
		default_func(key)
		inst.searchDoneHandler(key)
	})
}

func (inst *SearchableView) SetSearchChangedFunc(handler func(text string)) {
	inst.searchInput.SetChangedFunc(handler)
}

func (inst *SearchableView) GetSearchText() string {
	return inst.searchInput.GetText()
}

func (inst *SearchableView) SetSearchText(text string) {
	inst.searchInput.SetText(text)
	inst.searchDoneHandler(APP_KEY_BINDINGS.Done)
}

func (inst *SearchableView) IsEscapable() bool {
	return !inst.isSearchHidden
}
