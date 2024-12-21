package core

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServiceRootView struct {
	pages         *tview.Pages
	paginatorView PaginatorView
	pageIndex     *int
	RootView      *tview.Flex
	orderedPages  []string
	app           *tview.Application
}

func NewServiceRootView(
	app *tview.Application,
	serviceName string,
	pages *tview.Pages,
	orderedPages []string,
) *ServiceRootView {

	var paginatorView = CreatePaginatorView(serviceName)
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(paginatorView.RootView, 1, 0, false)

	var pageIndex = 0
	return &ServiceRootView{
		RootView:      rootView,
		pages:         pages,
		paginatorView: paginatorView,
		pageIndex:     &pageIndex,
		orderedPages:  orderedPages,
		app:           app,
	}
}

func (inst *ServiceRootView) Init() *ServiceRootView {
	inst.initPageNavigation()
	return inst
}

func (inst *ServiceRootView) ChangePage(pageIdx int, focusView tview.Primitive) {
	var numPages = len(inst.orderedPages)
	*inst.pageIndex = (pageIdx + numPages) % numPages
	var pageName = inst.orderedPages[*inst.pageIndex]
	inst.pages.SwitchToPage(pageName)
	if focusView != nil {
		inst.app.SetFocus(focusView)
	}
	inst.paginatorView.PageNameView.SetText(pageName)
	inst.paginatorView.PageCounterView.SetText(
		fmt.Sprintf("<%d/%d>", *inst.pageIndex+1, numPages),
	)
}

func (inst *ServiceRootView) initPageNavigation() {
	var numPages = len(inst.orderedPages)
	inst.ChangePage(0, nil)

	inst.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlH:
			*inst.pageIndex = (*inst.pageIndex - 1 + numPages) % numPages
			inst.ChangePage(*inst.pageIndex, nil)
			return nil
		case tcell.KeyCtrlL:
			*inst.pageIndex = (*inst.pageIndex + 1) % numPages
			inst.ChangePage(*inst.pageIndex, nil)
			return nil
		}
		return event
	})
}
