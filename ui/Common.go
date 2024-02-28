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

type pageInfoView struct {
	PageCounterView *tview.TextView
	RootView        *tview.Flex
}

func createPaginatorView() pageInfoView {

	var pageCount = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tertiaryTextColor)
	pageCount.SetBorderPadding(0, 0, 1, 1)
	var rootView = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(pageCount, 0, 1, false)
	return pageInfoView{
		PageCounterView: pageCount,
		RootView:        rootView,
	}
}

func initPageNavigation(
	app *tview.Application,
	pages *tview.Pages,
	pageIdx *int,
	orderedPageNames []string,
	paginationView *tview.TextView,
) {
	var numPages = len(orderedPageNames)
	paginationView.SetText(fmt.Sprintf("<%d/%d>", *pageIdx+1, numPages))

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlH:
			*pageIdx = (*pageIdx - 1 + numPages) % numPages
			var pageName = orderedPageNames[*pageIdx]
			pages.SwitchToPage(pageName)
			paginationView.SetText(fmt.Sprintf("<%d/%d>", *pageIdx+1, numPages))
			return nil
		case tcell.KeyCtrlL:
			*pageIdx = (*pageIdx + 1) % numPages
			var pageName = orderedPageNames[*pageIdx]
			pages.SwitchToPage(pageName)
			paginationView.SetText(fmt.Sprintf("<%d/%d>", *pageIdx+1, numPages))
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
