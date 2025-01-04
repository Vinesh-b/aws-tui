package core

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServiceRootView struct {
	*tview.Flex
	pages           *tview.Pages
	paginatorView   PaginatorView
	pageIndex       int
	orderedPages    []string
	pageViewMap     map[string]ServicePage
	lastFocusedView tview.Primitive
	app             *tview.Application
}

func NewServiceRootView(
	app *tview.Application,
	serviceName string,
) *ServiceRootView {
	var paginatorView = CreatePaginatorView(serviceName)

	var view = &ServiceRootView{
		Flex:            tview.NewFlex().SetDirection(tview.FlexRow),
		pages:           tview.NewPages(),
		paginatorView:   paginatorView,
		pageIndex:       0,
		orderedPages:    []string{},
		pageViewMap:     map[string]ServicePage{},
		lastFocusedView: nil,
		app:             app,
	}

	view.
		AddItem(view.pages, 0, 1, true).
		AddItem(paginatorView, 1, 0, false)

	return view
}

func (inst *ServiceRootView) ChangePage(pageIdx int, focusView tview.Primitive) {
	var numPages = len(inst.orderedPages)
	inst.pageIndex = (pageIdx + numPages) % numPages
	var pageName = inst.orderedPages[inst.pageIndex]
	inst.pages.SwitchToPage(pageName)
	if focusView != nil {
		inst.app.SetFocus(focusView)
	} else {
		if page, ok := inst.pageViewMap[pageName]; ok {
			inst.app.SetFocus(page.GetLastFocusedView())
		}
	}

	inst.paginatorView.PageNameView.SetText(pageName)
	inst.paginatorView.PageCounterView.SetText(
		fmt.Sprintf("<%d/%d>", inst.pageIndex+1, numPages),
	)
}

func (inst *ServiceRootView) InitPageNavigation() {
	var numPages = len(inst.orderedPages)
	inst.ChangePage(0, nil)

	inst.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.PageBack:
			inst.pageIndex = (inst.pageIndex - 1 + numPages) % numPages
			inst.ChangePage(inst.pageIndex, nil)
			return nil
		case APP_KEY_BINDINGS.PageForward:
			inst.pageIndex = (inst.pageIndex + 1) % numPages
			inst.ChangePage(inst.pageIndex, nil)
			return nil
		}
		return event
	})
}

func (inst *ServiceRootView) GetLastFocusedView() tview.Primitive {
	var pageName = inst.orderedPages[inst.pageIndex]
	return inst.pageViewMap[pageName].GetLastFocusedView()
}

func (inst *ServiceRootView) AddPage(
	name string, item ServicePage, resize bool, visible bool,
) *ServiceRootView {
	inst.pages.AddPage(name, item, resize, visible)
	inst.orderedPages = append(inst.orderedPages, name)
	inst.pageViewMap[name] = item
	return inst
}

func (inst *ServiceRootView) AddAndSwitchToPage(
	name string, item ServicePage, resize bool,
) *ServiceRootView {
	inst.pages.AddAndSwitchToPage(name, item, resize)
	inst.orderedPages = append(inst.orderedPages, name)
	inst.pageViewMap[name] = item
	return inst
}
