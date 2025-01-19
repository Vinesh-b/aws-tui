package core

import (
	"fmt"
	"log"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const SERVICE_PAGES_LIST = "SERVICE_PAGES_LIST"

type ServiceRootView struct {
	*tview.Flex
	pages           *tview.Pages
	paginatorView   PaginatorView
	pageIndex       int
	orderedPages    []string
	pageViewMap     map[string]ServicePage
	lastFocusedView tview.Primitive
	pagesListHidden bool
	pageList        *tview.List
	app             *tview.Application
}

func NewServiceRootView(
	serviceName string,
	app *tview.Application,
	config *aws.Config,
	logger *log.Logger,

) *ServiceRootView {
	var paginatorView = CreatePaginatorView(serviceName, app, config, logger)

	var view = &ServiceRootView{
		Flex:            tview.NewFlex().SetDirection(tview.FlexRow),
		pages:           tview.NewPages(),
		paginatorView:   paginatorView,
		pageIndex:       0,
		orderedPages:    []string{},
		pageViewMap:     map[string]ServicePage{},
		lastFocusedView: nil,
		pagesListHidden: true,
		pageList:        tview.NewList(),
		app:             app,
	}

	view.pageList.
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(TextColour).
		SetSelectedTextColor(InverseTextColor)

	view.pageList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var currentIdx = view.pageList.GetCurrentItem()
		var numItems = view.pageList.GetItemCount()
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case APP_KEY_BINDINGS.MoveUpRune:
				currentIdx = (currentIdx - 1 + numItems) % numItems
				view.pageList.SetCurrentItem(currentIdx)
				return nil
			case APP_KEY_BINDINGS.MoveDownRune:
				currentIdx = (currentIdx + 1) % numItems
				view.pageList.SetCurrentItem(currentIdx)
				return nil
			}
		}

		return event
	})

	view.pageList.SetSelectedFunc(func(i int, pageName, s2 string, r rune) {
		view.pages.HidePage(SERVICE_PAGES_LIST)
		view.pagesListHidden = true
		view.switchToPage(pageName)
	})

	view.pages.AddPage(SERVICE_PAGES_LIST,
		FloatingView("Pages", view.pageList, 30, 10),
		true, false,
	)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case APP_KEY_BINDINGS.Escape:
			if !view.pagesListHidden {
				view.pages.HidePage(SERVICE_PAGES_LIST)
				app.SetFocus(view.lastFocusedView)
				view.pagesListHidden = true
				return nil
			}
		case APP_KEY_BINDINGS.ToggleServicePages:
			if view.pagesListHidden {
				view.pages.ShowPage(SERVICE_PAGES_LIST)
				app.SetFocus(view.pageList)
			} else {
				view.pages.HidePage(SERVICE_PAGES_LIST)
				app.SetFocus(view.lastFocusedView)
			}
			view.pagesListHidden = !view.pagesListHidden
			return nil
		}
		return event
	})

	view.
		AddItem(view.pages, 0, 1, true).
		AddItem(paginatorView, 2, 0, false)

	return view
}

func (inst *ServiceRootView) switchToPage(name string) {
	inst.pages.SwitchToPage(name)
	if page, ok := inst.pageViewMap[name]; ok {
		inst.lastFocusedView = page.GetLastFocusedView()
		inst.app.SetFocus(inst.lastFocusedView)
	}

	var numPages = len(inst.orderedPages)
	var idx = slices.IndexFunc(
		inst.orderedPages,
		func(n string) bool { return n == name },
	)

	inst.pageIndex = max(idx, 0)
	inst.paginatorView.PageNameView.SetText(name)
	inst.paginatorView.PageCounterView.SetText(
		fmt.Sprintf("<%d/%d>", inst.pageIndex+1, numPages),
	)
}

func (inst *ServiceRootView) ChangePage(pageIdx int, focusView tview.Primitive) {
	var pageName = inst.orderedPages[min(inst.pageIndex, len(inst.orderedPages))]
	inst.switchToPage(pageName)
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
	inst.pageList.AddItem(name, "", 0, nil)
	inst.pages.SendToFront(SERVICE_PAGES_LIST)
	return inst
}

func (inst *ServiceRootView) AddAndSwitchToPage(
	name string, item ServicePage, resize bool,
) *ServiceRootView {
	inst.AddPage(name, item, resize, true)
	inst.pages.SwitchToPage(name)
	return inst
}
