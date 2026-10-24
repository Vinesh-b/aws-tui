package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	SEARCH_OVERLAY_NAME = "SEARCH"
)

type SearchableView struct {
	*BaseView
	HighlightSearch bool

	searchInput       *tview.InputField
	searchDoneHandler func(key tcell.Key)
	appCtx            *AppContext
}

func NewSearchableView(
	mainPage tview.Primitive,
	appContext *AppContext,
) *SearchableView {
	var floatingSearch = NewFloatingSearchView("Search", 0, 3)
	var view = &SearchableView{
		BaseView:        NewBaseView(appContext),
		HighlightSearch: false,

		searchInput:       floatingSearch.InputField,
		searchDoneHandler: func(key tcell.Key) {},
		appCtx:            appContext,
	}

	view.SetMainView(mainPage)
	view.AddRuneToggleOverlay(SEARCH_OVERLAY_NAME, floatingSearch, APP_KEY_BINDINGS.Find, false)
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
		case APP_KEY_BINDINGS.Done:
			inst.ToggleOverlay(SEARCH_OVERLAY_NAME, true)
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
	return !inst.IsOverlayHidden(SEARCH_OVERLAY_NAME)
}
