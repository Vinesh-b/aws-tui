package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/atotto/clipboard"
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

func createTextArea(title string) *tview.TextArea {
	var textArea = tview.NewTextArea().
		SetClipboard(
			func(s string) { clipboard.WriteAll(s) },
			func() string {
				var res, _ = clipboard.ReadAll()
				return res
			},
		).
		SetSelectedStyle(
			tcell.Style{}.Background(moreContrastBackgroundColor),
		)
	textArea.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	return textArea
}

type messageDataType int

const (
	DATA_TYPE_STRING messageDataType = iota
	DATA_TYPE_MAP_STRING_ANY
)

func createExpandedLogView(
	app *tview.Application,
	table *tview.Table,
	fixedColIdx int,
	dataType messageDataType,
) *tview.TextArea {
	var expandedView = createTextArea("Message")

	table.SetSelectionChangedFunc(func(row, column int) {
		var col = column
		if fixedColIdx >= 0 {
			col = fixedColIdx
		}

		var privateData = table.GetCell(row, col).Reference
		if row < 1 || privateData == nil {
			return
		}

		var anyJson map[string]interface{}
		var logText = ""

		switch dataType {
		case DATA_TYPE_STRING:
			var logText = privateData.(string)
			var err = json.Unmarshal([]byte(logText), &anyJson)
			if err != nil {
				expandedView.SetText(logText, false)
				return
			}
		case DATA_TYPE_MAP_STRING_ANY:
			anyJson = privateData.(map[string]interface{})
		}

		var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")
		logText = string(jsonBytes)
		expandedView.SetText(logText, false)
	})

	table.SetSelectedFunc(func(row, column int) {
		app.SetFocus(expandedView)
	})

	return expandedView
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

type ServiceView struct {
	RootView              *tview.Flex
	LastFocusedView       tview.Primitive
	orderedViews          []view
	viewResizeEnabled     bool
	topView               view
	bottomView            view
	topViewDefaultSize    int
	bottomViewDefaultSize int
	selectedViewIdx       int
	app                   *tview.Application
	logger                *log.Logger
}

func NewServiceView(
	app *tview.Application,
	logger *log.Logger,
) *ServiceView {
	var rootView = tview.NewFlex().SetDirection(tview.FlexRow)
	return &ServiceView{
		RootView:          rootView,
		LastFocusedView:   nil,
		viewResizeEnabled: false,
		app:               app,
		logger:            logger,
	}
}

func (inst *ServiceView) InitViewNavigation(orderedViews []view) {
	inst.orderedViews = orderedViews
	var viewIdx = 0
	var numViews = len(inst.orderedViews)
	inst.RootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlJ:
			viewIdx = (viewIdx - 1 + numViews) % numViews
			inst.LastFocusedView = inst.orderedViews[viewIdx]
			inst.app.SetFocus(inst.LastFocusedView)
			return nil
		case tcell.KeyCtrlK:
			viewIdx = (viewIdx + 1) % numViews
			inst.LastFocusedView = inst.orderedViews[viewIdx]
			inst.app.SetFocus(inst.LastFocusedView)
			return nil
		}

		if inst.viewResizeEnabled {
			event = inst.paneResizeHightHandler(event)
		}

		return event
	})
}

func (inst *ServiceView) InitViewTabNavigation(rootView rootView, orderedViews []view) {
	// Sets current view index when selected
	var viewIdx = -1
	var numViews = len(orderedViews)
	rootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBacktab:
			viewIdx = (viewIdx - 1 + numViews) % numViews
			inst.app.SetFocus(orderedViews[viewIdx])
			return nil
		case tcell.KeyTab:
			viewIdx = (viewIdx + 1) % numViews
			inst.app.SetFocus(orderedViews[viewIdx])
			return nil
		}

		return event
	})
}

func (inst *ServiceView) SetResizableViews(
	topView view, bottomView view,
	topDefaultSize int, bottomDefaultSize int,
) {
	inst.topView = topView
	inst.bottomView = bottomView
	inst.topViewDefaultSize = topDefaultSize
	inst.bottomViewDefaultSize = bottomDefaultSize
	inst.viewResizeEnabled = true
}

func (inst *ServiceView) paneResizeHightHandler(
	event *tcell.EventKey,
) *tcell.EventKey {
	var _, _, _, topSize = inst.topView.GetRect()
	var _, _, _, bottomSize = inst.bottomView.GetRect()
	switch event.Modifiers() {
	case tcell.ModAlt:
		switch event.Rune() {
		case rune('j'):
			if bottomSize > 0 {
				inst.RootView.ResizeItem(inst.topView, 0, topSize+1)
				inst.RootView.ResizeItem(inst.bottomView, 0, bottomSize-1)
			}
			return nil
		case rune('k'):
			if topSize > 0 {
				inst.RootView.ResizeItem(inst.topView, 0, topSize-1)
				inst.RootView.ResizeItem(inst.bottomView, 0, bottomSize+1)
			}
			return nil
		case rune('0'):
			inst.RootView.ResizeItem(inst.topView, 0, inst.topViewDefaultSize)
			inst.RootView.ResizeItem(inst.bottomView, 0, inst.bottomViewDefaultSize)
			return nil
		}
	}

	return event
}

func highlightTableSearch(
	app *tview.Application,
	table *tview.Table,
	search string,
	cols []int,
) []int {
	clearSearchHighlights(table)

	var foundPositions []int
	if len(search) > 0 {
		foundPositions = searchRefsInTable(table, cols, search)
		if len(foundPositions) > 0 {
			table.Select(foundPositions[0], 0)
		}
	}
	return foundPositions
}
