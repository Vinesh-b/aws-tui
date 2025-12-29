package core

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ResizableView struct {
	*tview.Flex
	view1           View
	view1Defaultlen int
	view2           View
	view2Defaultlen int
	direction       int
}

func NewResizableView(
	view1 View, view1Len int, view2 View, view2Len int, direction int,
) *ResizableView {
	var view = &ResizableView{
		Flex:            tview.NewFlex(),
		view1:           view1,
		view1Defaultlen: view1Len,
		view2:           view2,
		view2Defaultlen: view1Len,
		direction:       direction,
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
