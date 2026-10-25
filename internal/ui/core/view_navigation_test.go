package core

import (
	"context"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type nestedNavFixture struct {
	app      *tview.Application
	page     *ServicePageView
	tabView  *TabViewHorizontal
	inner1   *tview.Box
	inner2   *tview.Box
	plainTab *tview.Box
	tableA   *tview.Box
	tableB   *tview.Box
}

// Layout:
//
//	row 0: [ tabView: "Split" tab {inner1, inner2} | "Plain" tab {plainTab} ]
//	row 1: [ tableA, tableB ]
func newNestedNavFixture() *nestedNavFixture {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})

	var f = &nestedNavFixture{
		app:      app,
		inner1:   tview.NewBox(),
		inner2:   tview.NewBox(),
		plainTab: tview.NewBox(),
		tableA:   tview.NewBox(),
		tableB:   tview.NewBox(),
	}

	var split = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(f.inner1, 0, 1, true).
		AddItem(f.inner2, 0, 1, false)

	f.tabView = NewTabViewHorizontal(appCtx).
		AddTab("Plain", f.plainTab, 0, 1, true).
		AddAndSwitchToTab("Split", split, 0, 1, true)
	f.tabView.GetTab("Split").InitViewNavigation([][]View{{f.inner1, f.inner2}})

	var tables = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(f.tableA, 0, 1, true).
		AddItem(f.tableB, 0, 1, false)

	f.page = NewServicePageView(appCtx)
	f.page.MainPage.
		AddItem(f.tabView, 0, 1, true).
		AddItem(tables, 0, 1, false)
	f.page.InitViewNavigation([][]View{
		{f.tabView},
		{f.tableA, f.tableB},
	})

	return f
}

func (f *nestedNavFixture) press(key tcell.Key) {
	var event = tcell.NewEventKey(key, 0, tcell.ModNone)
	f.page.InputHandler()(event, func(p tview.Primitive) { f.app.SetFocus(p) })
}

func (f *nestedNavFixture) assertFocus(t *testing.T, expected tview.Primitive, name string) {
	t.Helper()
	if f.app.GetFocus() != expected {
		t.Fatalf("Expected focus on %s", name)
	}
}

func TestViewNavigation2D__MovesWithinNestedTab(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.inner1)

	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.assertFocus(t, f.inner2, "inner2")

	f.press(APP_KEY_BINDINGS.ViewFocusLeft)
	f.assertFocus(t, f.inner1, "inner1")
}

func TestViewNavigation2D__ExitsNestedTabAtEdge(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.inner2)

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.tableA, "tableA")
}

func TestViewNavigation2D__HorizontalEdgeWrapsOuterRow(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.inner2)

	// The tab view is alone on its outer row so it wraps back into itself,
	// entering from the left edge
	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.assertFocus(t, f.inner1, "inner1")
}

func TestViewNavigation2D__EntersNestedTabAtLastFocusedView(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.inner1)

	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.tableA, "tableA")

	f.press(APP_KEY_BINDINGS.ViewFocusUp)
	f.assertFocus(t, f.inner2, "inner2")
}

func TestViewNavigation2D__EntersNestedTabFromWrap(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.tableB)

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.inner1, "inner1")
}

func TestViewNavigation2D__TabWithoutNavigationFocusesTabContent(t *testing.T) {
	var f = newNestedNavFixture()
	f.tabView.SwitchToTab("Plain")
	f.app.SetFocus(f.tableA)

	f.press(APP_KEY_BINDINGS.ViewFocusUp)
	f.assertFocus(t, f.plainTab, "plainTab")

	f.press(APP_KEY_BINDINGS.ViewFocusDown)
	f.assertFocus(t, f.tableA, "tableA")
}

func TestViewNavigation2D__OuterNavigationWraps(t *testing.T) {
	var f = newNestedNavFixture()
	f.app.SetFocus(f.tableB)

	f.press(APP_KEY_BINDINGS.ViewFocusRight)
	f.assertFocus(t, f.tableA, "tableA")

	f.press(APP_KEY_BINDINGS.ViewFocusLeft)
	f.assertFocus(t, f.tableB, "tableB")
}

func TestViewNavigation2D__NavigateWithinStopsAtEdges(t *testing.T) {
	var app = tview.NewApplication()
	var a, b, c = tview.NewBox(), tview.NewBox(), tview.NewBox()
	var flex = tview.NewFlex().
		AddItem(a, 0, 1, true).
		AddItem(b, 0, 1, false).
		AddItem(c, 0, 1, false)

	var nav = NewViewNavigation2D(flex, nil, app)
	nav.UpdateOrderedViews([][]View{{a, b}, {c}}, 0)
	app.SetFocus(a)

	if nav.NavigateWithin(APP_KEY_BINDINGS.ViewFocusLeft) || app.GetFocus() != a {
		t.Fatalf("Expected left edge to be reported and focus to stay on a")
	}
	if nav.NavigateWithin(APP_KEY_BINDINGS.ViewFocusUp) || app.GetFocus() != a {
		t.Fatalf("Expected top edge to be reported and focus to stay on a")
	}
	if !nav.NavigateWithin(APP_KEY_BINDINGS.ViewFocusRight) || app.GetFocus() != b {
		t.Fatalf("Expected focus to move to b")
	}
	if !nav.NavigateWithin(APP_KEY_BINDINGS.ViewFocusDown) || app.GetFocus() != c {
		t.Fatalf("Expected focus to move down to c, clamped to the shorter row")
	}
	if nav.NavigateWithin(APP_KEY_BINDINGS.ViewFocusDown) || app.GetFocus() != c {
		t.Fatalf("Expected bottom edge to be reported and focus to stay on c")
	}
}

func TestViewNavigation2D__NavigatesServicePageAddedAsTab(t *testing.T) {
	var app = tview.NewApplication()
	var appCtx = NewAppContext(context.Background(), app, nil, nil, &AppTheme{})
	var input, output, list = tview.NewBox(), tview.NewBox(), tview.NewBox()

	// A tab whose content is a page with its own navigation, like the lambda
	// invoke view
	var innerPage = NewServicePageView(appCtx)
	innerPage.MainPage.SetDirection(tview.FlexColumn).
		AddItem(input, 0, 1, true).
		AddItem(output, 0, 1, false)
	innerPage.InitViewNavigation([][]View{{input, output}})

	var tabView = NewTabViewHorizontal(appCtx).
		AddAndSwitchToTab("Invoke", innerPage, 0, 1, true)

	var page = NewServicePageView(appCtx)
	page.MainPage.
		AddItem(tabView, 0, 1, true).
		AddItem(list, 0, 1, false)
	page.InitViewNavigation([][]View{{tabView}, {list}})

	var press = func(key tcell.Key) {
		var event = tcell.NewEventKey(key, 0, tcell.ModNone)
		page.InputHandler()(event, func(p tview.Primitive) { app.SetFocus(p) })
	}

	app.SetFocus(list)
	press(APP_KEY_BINDINGS.ViewFocusUp)
	if app.GetFocus() != input {
		t.Fatalf("Expected focus on input when entering the tab")
	}

	press(APP_KEY_BINDINGS.ViewFocusRight)
	if app.GetFocus() != output {
		t.Fatalf("Expected focus on output within the tab")
	}

	press(APP_KEY_BINDINGS.ViewFocusDown)
	if app.GetFocus() != list {
		t.Fatalf("Expected focus on list after leaving the tab")
	}
}
