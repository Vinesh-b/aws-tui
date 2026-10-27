package core

import (
	"slices"
	"testing"

	"github.com/rivo/tview"
)

type boxPage struct{ *tview.Box }

func (p boxPage) GetLastFocusedView() tview.Primitive { return p.Box }

// newPagedRootView has the pages Groups and Streams, and reports each page
// that is shown after it is built
func newPagedRootView(t *testing.T) (*ServiceRootView, *[]string) {
	var view = NewServiceRootView("Logs", newTestAppContext())
	t.Cleanup(view.Close)
	view.
		AddAndSwitchToPage("Groups", boxPage{tview.NewBox()}, true).
		AddPage("Streams", boxPage{tview.NewBox()}, true, true)
	view.InitPageNavigation()

	var shown = &[]string{}
	view.SetServicePageChangedFunc(func(name string) { *shown = append(*shown, name) })
	return view, shown
}

func TestServiceRootView__ReportsThePagesThatAreShown(t *testing.T) {
	var view, shown = newPagedRootView(t)

	view.ChangePage(1, nil)
	view.ShowServicePage("Groups")

	if !slices.Equal(*shown, []string{"Streams", "Groups"}) {
		t.Errorf("pages shown = %v, want [Streams Groups]", *shown)
	}
	if got := view.CurrentServicePage(); got != "Groups" {
		t.Errorf("current page = %q, want Groups", got)
	}
}

func TestServiceRootView__ShowingAPageItDoesNotHaveChangesNothing(t *testing.T) {
	var view, shown = newPagedRootView(t)
	view.ChangePage(1, nil)

	view.ShowServicePage("Events")

	if got := view.CurrentServicePage(); got != "Streams" {
		t.Errorf("current page = %q, want Streams", got)
	}
	if len(*shown) != 1 {
		t.Errorf("pages shown = %v, want only Streams", *shown)
	}
}

func TestServiceRootView__WithoutPagesHasNoCurrentPage(t *testing.T) {
	var view = NewServiceRootView("Empty", newTestAppContext())
	t.Cleanup(view.Close)

	if got := view.CurrentServicePage(); got != "" {
		t.Errorf("current page = %q, want none", got)
	}
}
