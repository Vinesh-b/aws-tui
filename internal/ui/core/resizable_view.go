package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ResizableView splits its space between two views, whose sizes the resize
// keys change. The focus keys along the split move between the two views when
// it is inside another view's navigation, such as a tab.
type ResizableView struct {
	*tview.Flex
	view1           View
	view1Defaultlen int
	view2           View
	view2Defaultlen int
	direction       int
	// focusedView is the pane focused last, which is focused again when the
	// view is entered from a side that doesn't pick one
	focusedView View
	appCtx      *AppContext
}

func NewResizableView(
	view1 View, view1Len int, view2 View, view2Len int, direction int, appCtx *AppContext,
) *ResizableView {
	var view = &ResizableView{
		Flex:            tview.NewFlex(),
		view1:           view1,
		view1Defaultlen: view1Len,
		view2:           view2,
		view2Defaultlen: view2Len,
		direction:       direction,
		focusedView:     view1,
		appCtx:          appCtx,
	}
	view.SetDirection(direction).
		AddItem(view1, 0, view1Len, true).
		AddItem(view2, 0, view2Len, true)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		event = view.paneResizeHightHandler(event)
		return event
	})

	return view
}

func (inst *ResizableView) paneResizeHightHandler(event *tcell.EventKey) *tcell.EventKey {
	var _, _, view1wSize, view1hSize = inst.view1.GetRect()
	var _, _, view2wSize, view2hSize = inst.view2.GetRect()
	var view1Size, view2Size int
	var movePositiveKey, moveNegativeKey rune

	if inst.direction == tview.FlexRow {
		view1Size = view1hSize
		view2Size = view2hSize
		movePositiveKey = APP_KEY_BINDINGS.MoveDownRune
		moveNegativeKey = APP_KEY_BINDINGS.MoveUpRune
	} else {
		view1Size = view1wSize
		view2Size = view2wSize
		movePositiveKey = APP_KEY_BINDINGS.MoveRightRune
		moveNegativeKey = APP_KEY_BINDINGS.MoveLeftRune
	}

	switch event.Modifiers() {
	case APP_KEY_BINDINGS.ViewResizeModKey:
		switch event.Rune() {
		case movePositiveKey:
			if view2Size > 0 {
				inst.ResizeItem(inst.view1, 0, view1Size+1)
				inst.ResizeItem(inst.view2, 0, view2Size-1)
			}
			return nil
		case moveNegativeKey:
			if view1Size > 0 {
				inst.ResizeItem(inst.view1, 0, view1Size-1)
				inst.ResizeItem(inst.view2, 0, view2Size+1)
			}
			return nil
		case APP_KEY_BINDINGS.ViewResizeReset:
			inst.ResizeItem(inst.view1, 0, inst.view1Defaultlen)
			inst.ResizeItem(inst.view2, 0, inst.view2Defaultlen)
			// Propagate reset command to all views
			return event
		}
	}

	return event
}

// paneKeys are the focus keys that move to the first and the second pane,
// which are those along the direction of the split
func (inst *ResizableView) paneKeys() (tcell.Key, tcell.Key) {
	if inst.direction == tview.FlexRow {
		return APP_KEY_BINDINGS.ViewFocusUp, APP_KEY_BINDINGS.ViewFocusDown
	}
	return APP_KEY_BINDINGS.ViewFocusLeft, APP_KEY_BINDINGS.ViewFocusRight
}

func (inst *ResizableView) focusPane(pane View, key tcell.Key) {
	inst.focusedView = pane
	if nested, ok := pane.(NestedNavigation); ok {
		nested.FocusFrom(key)
		return
	}
	inst.appCtx.App.SetFocus(pane)
}

// NavigateWithin moves focus within the focused pane, or to the other pane
// with the focus keys along the split. It is false at the edge of the split
// and for the other keys, so the parent navigation moves focus instead.
func (inst *ResizableView) NavigateWithin(key tcell.Key) bool {
	var focused, other = inst.view1, inst.view2
	if inst.view2.HasFocus() {
		focused, other = inst.view2, inst.view1
	}
	inst.focusedView = focused

	if nested, ok := focused.(NestedNavigation); ok && nested.NavigateWithin(key) {
		return true
	}

	var toFirst, toSecond = inst.paneKeys()
	if (focused == inst.view1 && key == toSecond) || (focused == inst.view2 && key == toFirst) {
		inst.focusPane(other, key)
		return true
	}
	return false
}

// FocusFrom focuses the pane nearest the side being entered from, or the pane
// focused last when entered across the split or without a key.
func (inst *ResizableView) FocusFrom(key tcell.Key) {
	var toFirst, toSecond = inst.paneKeys()
	switch key {
	case toSecond:
		inst.focusPane(inst.view1, key)
	case toFirst:
		inst.focusPane(inst.view2, key)
	default:
		inst.focusPane(inst.focusedView, key)
	}
}
