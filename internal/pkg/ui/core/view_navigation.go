package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ViewNavigation struct {
	app          *tview.Application
	rootView     RootView
	orderedViews []View
	viewIdx      int
	numViews     int
	keyForward   tcell.Key
	keyBack      tcell.Key
}

func NewViewNavigation(rootView RootView, orderedViews []View, app *tview.Application) *ViewNavigation {
	var view = &ViewNavigation{
		rootView:     rootView,
		orderedViews: orderedViews,
		app:          app,
		viewIdx:      len(orderedViews),
		numViews:     len(orderedViews),
		keyForward:   APP_KEY_BINDINGS.FormFocusNext,
		keyBack:      APP_KEY_BINDINGS.FormFocusPrev,
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case view.keyForward:
			view.viewIdx = (view.viewIdx - 1 + view.numViews) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		case view.keyBack:
			view.viewIdx = (view.viewIdx + 1) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		}

		return event
	})

	return view
}

func (inst *ViewNavigation) SetNavigationKeys(keyForward tcell.Key, keyBack tcell.Key) {
	inst.keyForward = keyForward
	inst.keyBack = keyBack
}

func (inst *ViewNavigation) UpdateOrderedViews(orderedViews []View, intitalIxd int) {
	inst.orderedViews = orderedViews
	inst.numViews = len(inst.orderedViews)
	inst.viewIdx = (intitalIxd + inst.numViews) % inst.numViews
}

func (inst *ViewNavigation) GetLastFocusedView() tview.Primitive {
	return inst.orderedViews[inst.viewIdx]
}
