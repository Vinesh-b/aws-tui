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
	var view = NewResizableView(left, 1, right, 3, tview.FlexColumn, newTestAppContext())
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

type splitTabFixture struct {
	app   *tview.Application
	page  *ServicePageView
	left  *tview.Box
	right *tview.Box
	top   *tview.Box
	below *tview.Box
	table *tview.Box
}

// Layout:
//
//	row 0: [ tabView: "Split" tab {left | right}, "Rows" tab {top / below} ]
//	row 1: [ table ]
func newSplitTabFixture() *splitTabFixture {
	var appCtx = newTestAppContext()
	var f = &splitTabFixture{
		app:   appCtx.App,
		left:  tview.NewBox(),
		right: tview.NewBox(),
		top:   tview.NewBox(),
		below: tview.NewBox(),
		table: tview.NewBox(),
	}

	var tabView = NewTabViewHorizontal(appCtx).
		AddAndSwitchToTab("Split", NewResizableView(f.left, 1, f.right, 1, tview.FlexColumn, appCtx), 0, 1, true).
		AddTab("Rows", NewResizableView(f.top, 1, f.below, 1, tview.FlexRow, appCtx), 0, 1, true)

	f.page = NewServicePageView(appCtx)
	f.page.MainPage.
		AddItem(tabView, 0, 1, true).
		AddItem(f.table, 0, 1, false)
	f.page.InitViewNavigation([][]View{{tabView}, {f.table}})

	return f
}

func (f *splitTabFixture) press(key tcell.Key) {
	var event = tcell.NewEventKey(key, 0, tcell.ModNone)
	f.page.InputHandler()(event, func(p tview.Primitive) { f.app.SetFocus(p) })
}

func (f *splitTabFixture) assertFocus(t *testing.T, expected tview.Primitive, name string) {
	t.Helper()
	if f.app.GetFocus() != expected {
		t.Fatalf("expected focus on %s", name)
	}
}

func TestResizableView__FocusKeysMoveBetweenThePanesOfATab(t *testing.T) {
	var f = newSplitTabFixture()
	f.app.SetFocus(f.left)

	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.assertFocus(t, f.right, "the right pane")

	f.press(APP_KEY_BINDINGS.ViewFocusLeft)
	f.assertFocus(t, f.left, "the left pane")
}

func TestResizableView__KeysAcrossTheSplitLeaveTheTab(t *testing.T) {
	var f = newSplitTabFixture()
	f.app.SetFocus(f.right)

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.table, "the table below the tabs")

	// Coming back focuses the pane that was focused last
	f.press(APP_KEY_BINDINGS.ViewFocusUp)
	f.assertFocus(t, f.right, "the right pane")
}

func TestResizableView__AtTheEdgeTheParentMovesFocus(t *testing.T) {
	var f = newSplitTabFixture()
	f.app.SetFocus(f.right)

	// The page has one column, so it wraps back into the tab from the left
	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.assertFocus(t, f.left, "the left pane")
}

func TestResizableView__RowSplitMovesUpAndDownWithin(t *testing.T) {
	var f = newSplitTabFixture()
	var tabView = f.page.GetViewNavigation()[0][0].(*TabViewHorizontal)
	tabView.SwitchToTab("Rows")
	f.app.SetFocus(f.top)

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.below, "the lower pane")

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.table, "the table below the tabs")

	// Entering from below focuses the lower pane
	f.press(APP_KEY_BINDINGS.ViewFocusUp)
	f.assertFocus(t, f.below, "the lower pane")
}
