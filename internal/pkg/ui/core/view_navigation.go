package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ViewNavigation1D struct {
	app          *tview.Application
	rootView     RootView
	orderedViews []View
	viewIdx      int
	numViews     int
	keyForward   tcell.Key
	keyBack      tcell.Key
}

func NewViewNavigation1D(
	rootView RootView, orderedViews []View, app *tview.Application,
) *ViewNavigation1D {
	var view = &ViewNavigation1D{
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

func (inst *ViewNavigation1D) SetNavigationKeys(keyForward tcell.Key, keyBack tcell.Key) {
	inst.keyForward = keyForward
	inst.keyBack = keyBack
}

func (inst *ViewNavigation1D) UpdateOrderedViews(orderedViews []View, intitalIxd int) {
	inst.orderedViews = orderedViews
	inst.numViews = len(inst.orderedViews)
	inst.viewIdx = (intitalIxd + inst.numViews) % inst.numViews
}

func (inst *ViewNavigation1D) GetOrderedViews() []View {
	return inst.orderedViews
}

func (inst *ViewNavigation1D) GetLastFocusedView() tview.Primitive {
	if len(inst.orderedViews) == 0 {
		return nil
	}
	return inst.orderedViews[inst.viewIdx]
}
