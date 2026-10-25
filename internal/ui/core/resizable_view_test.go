package core

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func drawResizableView(t *testing.T, view *ResizableView) {
	t.Helper()
	var screen = newTestScreen(t, 40, 10)
	view.SetRect(0, 0, 40, 10)
	view.Draw(screen)
}

func widthOf(view View) int {
	var _, _, width, _ = view.GetRect()
	return width
}

func TestResizableView__ResetRestoresUnevenDefaultSplit(t *testing.T) {
	var left, right = tview.NewBox(), tview.NewBox()
	var view = NewResizableView(left, 1, right, 3, tview.FlexColumn)
	drawResizableView(t, view)

	var handler = view.InputHandler()
	var resizeKey = func(r rune) *tcell.EventKey {
		return tcell.NewEventKey(tcell.KeyRune, r, APP_KEY_BINDINGS.ViewResizeModKey)
	}

	handler(resizeKey(APP_KEY_BINDINGS.MoveRightRune), func(tview.Primitive) {})
	drawResizableView(t, view)
	if got := widthOf(left); got == 10 {
		t.Fatalf("left width = %d, expected resize to change it", got)
	}

	handler(resizeKey(APP_KEY_BINDINGS.ViewResizeReset), func(tview.Primitive) {})
	drawResizableView(t, view)

	if gotLeft, gotRight := widthOf(left), widthOf(right); gotLeft != 10 || gotRight != 30 {
		t.Errorf("widths after reset = %d/%d, want 10/30", gotLeft, gotRight)
	}
}
