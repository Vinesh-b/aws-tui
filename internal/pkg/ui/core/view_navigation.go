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
		viewIdx:      0,
		numViews:     len(orderedViews),
		keyForward:   APP_KEY_BINDINGS.FormFocusNext,
		keyBack:      APP_KEY_BINDINGS.FormFocusPrev,
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case view.keyForward:
			view.viewIdx = (view.viewIdx + 1) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		case view.keyBack:
			view.viewIdx = (view.viewIdx - 1 + view.numViews) % view.numViews
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

type ViewNavigation2D struct {
	app          *tview.Application
	rootView     RootView
	orderedViews [][]View
	colIdx       int
	numCol       int
	rowIdx       int
	numRow       int
	keyUp        tcell.Key
	keyDown      tcell.Key
	keyLeft      tcell.Key
	keyRight     tcell.Key
}

func NewViewNavigation2D(
	rootView RootView, orderedViews [][]View, app *tview.Application,
) *ViewNavigation2D {
	var view = &ViewNavigation2D{
		rootView:     rootView,
		orderedViews: orderedViews,
		app:          app,
		rowIdx:       0,
		numRow:       1,
		colIdx:       0,
		numCol:       1,
		keyUp:        APP_KEY_BINDINGS.ViewFocusUp,
		keyDown:      APP_KEY_BINDINGS.ViewFocusDown,
		keyLeft:      APP_KEY_BINDINGS.ViewFocusLeft,
		keyRight:     APP_KEY_BINDINGS.ViewFocusRight,
	}
	if len(view.orderedViews) > 0 {
		view.numCol = len(orderedViews[0])
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if view.orderedViews == nil || len(view.orderedViews) == 0 {
			return event
		}

		switch event.Key() {
		case view.keyUp:
			view.rowIdx = (view.rowIdx - 1 + view.numRow) % view.numRow
			var colIdx = view.colIdx
			if colIdx >= len(view.orderedViews[view.rowIdx]) {
				colIdx = len(view.orderedViews[view.rowIdx]) - 1
			}
			view.app.SetFocus(view.orderedViews[view.rowIdx][colIdx])
			return nil
		case view.keyDown:
			view.rowIdx = (view.rowIdx + 1) % view.numRow
			var colIdx = view.colIdx
			if colIdx >= len(view.orderedViews[view.rowIdx]) {
				colIdx = len(view.orderedViews[view.rowIdx]) - 1
			}
			view.app.SetFocus(view.orderedViews[view.rowIdx][colIdx])
			return nil
		case view.keyRight:
			view.numCol = len(view.orderedViews[view.rowIdx])
			view.colIdx = (view.colIdx + 1) % view.numCol
			view.app.SetFocus(view.orderedViews[view.rowIdx][view.colIdx])
			return nil
		case view.keyLeft:
			view.numCol = len(view.orderedViews[view.rowIdx])
			view.colIdx = (view.colIdx - 1 + view.numCol) % view.numCol
			view.app.SetFocus(view.orderedViews[view.rowIdx][view.colIdx])
			return nil
		}

		return event
	})

	return view
}

func (inst *ViewNavigation2D) UpdateOrderedViews(orderedViews [][]View, intitalIxd int) {
	if len(orderedViews) != 0 && len(orderedViews[0]) != 0 {
		inst.orderedViews = orderedViews
		inst.numRow = len(inst.orderedViews)
		inst.numCol = len(inst.orderedViews[0])
		inst.rowIdx = inst.numRow - 1
		inst.colIdx = 0
	}
}

func (inst *ViewNavigation2D) GetOrderedViews() [][]View {
	return inst.orderedViews
}

func (inst *ViewNavigation2D) GetLastFocusedView() tview.Primitive {
	if len(inst.orderedViews) == 0 || len(inst.orderedViews[inst.rowIdx]) == 0 {
		return nil
	}

	var row = inst.rowIdx % len(inst.orderedViews)
	var col = inst.colIdx % len(inst.orderedViews[row])
	return inst.orderedViews[row][col]
}
