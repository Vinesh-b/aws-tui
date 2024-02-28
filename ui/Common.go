package ui

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func loadData(
	app *tview.Application,
	view *tview.Box,
	resultChannel chan struct{},
	updateViewFunc func(),
) {
	var (
		idx           = 0
		originalTitle = view.GetTitle()
		loadingSymbol = [8]string{"⢎⡰", "⢎⡡", "⢎⡑", "⢎⠱", "⠎⡱", "⢊⡱", "⢌⡱", "⢆⡱"}

		timeoutCtx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	)
	defer cancel()

	for {
		select {
		case <-resultChannel:
			app.QueueUpdateDraw(updateViewFunc)
			return
		case <-timeoutCtx.Done():
			app.QueueUpdateDraw(func() {
				view.SetTitle("Timed out")
			})
			return
		default:
			app.QueueUpdateDraw(func() {
				view.SetTitle(fmt.Sprintf(originalTitle+"%s", loadingSymbol[idx]))
			})
			idx = (idx + 1) % len(loadingSymbol)
			time.Sleep(time.Millisecond * 100)
		}
	}
}

type tableCreationParams struct {
	App    *tview.Application
	Logger *log.Logger
}

type rootView interface {
	tview.Primitive
	SetInputCapture(callback func(*tcell.EventKey) *tcell.EventKey) *tview.Box
}

type view interface {
	tview.Primitive
	SetFocusFunc(callback func()) *tview.Box
}

func createSearchInput(label string) *tview.InputField {
	var inputField = tview.NewInputField().
		SetLabel(fmt.Sprintf("%s ", label)).
		SetFieldWidth(0)
	inputField.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle("Search").
		SetTitleAlign(tview.AlignLeft)

	return inputField
}

func initViewNavigation(
	app *tview.Application,
	rootView rootView,
	viewIdx *int,
	orderedViews []view,
) {
	// Sets current view index when selected
	for i, v := range orderedViews {
		v.SetFocusFunc(func() { *viewIdx = i })
	}

	var numViews = len(orderedViews)
	rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlJ:
			*viewIdx = (*viewIdx - 1 + numViews) % numViews
			app.SetFocus(orderedViews[*viewIdx])
			return nil
		case tcell.KeyCtrlK:
			*viewIdx = (*viewIdx + 1) % numViews
			app.SetFocus(orderedViews[*viewIdx])
			return nil
		}
		return event
	})
}

type paginatorView struct {
	PageCounterView *tview.TextView
	PageNameView    *tview.TextView
	RootView        *tview.Flex
}

func createPaginatorView(service string) paginatorView {
	var pageCount = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetTextColor(tertiaryTextColor)

	var pageName = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tertiaryTextColor)

	var serviceName = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tertiaryTextColor).
		SetText(service)

	var rootView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(serviceName, 0, 1, false).
		AddItem(pageName, 0, 1, false).
		AddItem(pageCount, 0, 1, false)
	rootView.SetBorderPadding(0, 0, 1, 1)

	return paginatorView{
		PageCounterView: pageCount,
		PageNameView:    pageName,
		RootView:        rootView,
	}
}

type ServiceRootView struct {
	pages         *tview.Pages
	paginatorView paginatorView
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

	var paginatorView = createPaginatorView(serviceName)
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

func highlightTableSearch(
	app *tview.Application,
	table *tview.Table,
	search string,
	cols []int,
) {
	var resultChannel = make(chan struct{})
	go func() {
		resultChannel <- struct{}{}
	}()
	go loadData(app, table.Box, resultChannel, func() {
		if len(search) == 0 {
			clearSearchHighlights(table)
		} else {
			searchRefsInTable(table, cols, search)
		}
	})
}
