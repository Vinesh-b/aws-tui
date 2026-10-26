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

	// Focus can be changed by a mouse selection bypassing the input capture handler
	var currentFocus = func(views []View) int {
		for idx, v := range views {
			if v.HasFocus() {
				return idx
			}
		}

		return 0
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case view.keyForward:
			view.viewIdx = currentFocus(view.orderedViews)
			view.viewIdx = (view.viewIdx + 1) % view.numViews
			view.app.SetFocus(view.orderedViews[view.viewIdx])
			return nil
		case view.keyBack:
			view.viewIdx = currentFocus(view.orderedViews)
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

// NestedNavigation is implemented by views that contain their own navigation,
// such as a tab view, allowing a parent navigation to move focus into, within
// and out of them.
type NestedNavigation interface {
	// NavigateWithin moves focus inside the view. Returns false when focus is
	// already at the edge in that direction so the parent navigation can move.
	NavigateWithin(key tcell.Key) bool
	// FocusFrom focuses the inner view nearest the edge being entered from.
	FocusFrom(key tcell.Key)
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
		view.numRow = len(orderedViews)
		view.numCol = len(orderedViews[0])
	}

	view.rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if len(view.orderedViews) == 0 {
			return event
		}

		switch event.Key() {
		case view.keyUp, view.keyDown, view.keyLeft, view.keyRight:
			var rowIdx, colIdx = view.currentFocus()
			var focused = view.orderedViews[rowIdx][colIdx]
			if nested, ok := focused.(NestedNavigation); ok && nested.NavigateWithin(event.Key()) {
				return nil
			}
			view.move(event.Key(), true)
			return nil
		}

		return event
	})

	return view
}

// Focus can be changed by a mouse selection bypassing the input capture handler
func (inst *ViewNavigation2D) currentFocus() (int, int) {
	for rowIdx, row := range inst.orderedViews {
		for colIdx, v := range row {
			if v.HasFocus() {
				return rowIdx, colIdx
			}
		}
	}

	return 0, 0
}

// move shifts focus one view in the direction of key. When wrap is false and
// focus is at the edge in that direction, focus is left unchanged and false is
// returned.
func (inst *ViewNavigation2D) move(key tcell.Key, wrap bool) bool {
	var rowIdx, colIdx = inst.currentFocus()
	var nextRow, nextCol = rowIdx, colIdx

	switch key {
	case inst.keyUp:
		nextRow--
	case inst.keyDown:
		nextRow++
	case inst.keyLeft:
		nextCol--
	case inst.keyRight:
		nextCol++
	default:
		return false
	}

	if nextRow < 0 || nextRow >= inst.numRow {
		if !wrap {
			return false
		}
		nextRow = (nextRow + inst.numRow) % inst.numRow
	}

	var rowLen = len(inst.orderedViews[nextRow])
	if nextRow != rowIdx {
		nextCol = min(nextCol, rowLen-1)
	} else if nextCol < 0 || nextCol >= rowLen {
		if !wrap {
			return false
		}
		nextCol = (nextCol + rowLen) % rowLen
	}

	inst.rowIdx, inst.colIdx = nextRow, nextCol
	inst.numCol = rowLen
	inst.focus(inst.orderedViews[nextRow][nextCol], key)
	return true
}

func (inst *ViewNavigation2D) focus(view View, key tcell.Key) {
	if nested, ok := view.(NestedNavigation); ok {
		nested.FocusFrom(key)
		return
	}
	inst.app.SetFocus(view)
}

func (inst *ViewNavigation2D) NavigateWithin(key tcell.Key) bool {
	if len(inst.orderedViews) == 0 {
		return false
	}

	var rowIdx, colIdx = inst.currentFocus()
	var focused = inst.orderedViews[rowIdx][colIdx]
	if nested, ok := focused.(NestedNavigation); ok && nested.NavigateWithin(key) {
		return true
	}

	return inst.move(key, false)
}

func (inst *ViewNavigation2D) FocusFrom(key tcell.Key) {
	if len(inst.orderedViews) == 0 {
		inst.app.SetFocus(inst.rootView)
		return
	}

	var rowIdx = inst.rowIdx % inst.numRow
	switch key {
	case inst.keyDown:
		rowIdx = 0
	case inst.keyUp:
		rowIdx = inst.numRow - 1
	}

	var rowLen = len(inst.orderedViews[rowIdx])
	var colIdx = min(inst.colIdx, rowLen-1)
	switch key {
	case inst.keyRight:
		colIdx = 0
	case inst.keyLeft:
		colIdx = rowLen - 1
	}

	inst.rowIdx, inst.colIdx = rowIdx, colIdx
	inst.numCol = rowLen
	inst.focus(inst.orderedViews[rowIdx][colIdx], key)
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

// SetFocusedRow makes the view in the first column of the row the one that
// is focused first, which is otherwise in the last row.
func (inst *ViewNavigation2D) SetFocusedRow(row int) {
	if row < 0 || row >= len(inst.orderedViews) || len(inst.orderedViews[row]) == 0 {
		return
	}
	inst.rowIdx = row
	inst.colIdx = 0
	inst.numCol = len(inst.orderedViews[row])
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
