package core

import (
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type RootView interface {
	tview.Primitive
	SetInputCapture(callback func(*tcell.EventKey) *tcell.EventKey) *tview.Box
}

type View interface {
	tview.Primitive
	SetFocusFunc(callback func()) *tview.Box
	SetTitle(title string) *tview.Box
	GetTitle() string
}

type ServicePage interface {
	tview.Primitive
	GetLastFocusedView() tview.Primitive
}

type MessagePromptType string

const (
	InfoPrompt    MessagePromptType = "INFO"
	ErrorPrompt   MessagePromptType = "ERROR"
	WarningPrompt MessagePromptType = "WARNING"
	DebugPrompt   MessagePromptType = "DEBUG"
)

type ServicePageView struct {
	*tview.Pages
	MainPage       *tview.Flex
	viewNavigation *ViewNavigation2D
	errorView      *MessagePromptView
	infoView       *MessagePromptView
	appCtx         *AppContext
	// messagesMtx guards lastMessages, the last message of each type, which
	// are given on any goroutine and shown on the UI goroutine
	messagesMtx  sync.Mutex
	lastMessages map[MessagePromptType]string
}

func NewServicePageView(appCtx *AppContext) *ServicePageView {
	var flex = tview.NewFlex()
	var viewNav = NewViewNavigation2D(flex, nil, appCtx.App)

	var view = &ServicePageView{
		MainPage:       flex,
		Pages:          tview.NewPages(),
		errorView:      NewMessagePromptView(appCtx.App),
		infoView:       NewMessagePromptView(appCtx.App),
		viewNavigation: viewNav,
		appCtx:         appCtx,
		lastMessages:   map[MessagePromptType]string{},
	}

	view.MainPage.SetDirection(tview.FlexRow)

	var floatingErrorView = FloatingView("Error", view.errorView, 80, 15)
	var floatingInfoView = FloatingView("Info", view.infoView, 80, 15)
	view.Pages.
		AddPage("MAIN_PAGE", view.MainPage, true, true).
		AddPage(string(ErrorPrompt), floatingErrorView, true, false).
		AddPage(string(InfoPrompt), floatingInfoView, true, false)

	view.errorView.SetSelectedFunc(func() {
		view.Pages.HidePage(string(ErrorPrompt))
		view.appCtx.App.SetFocus(view.GetLastFocusedView())
	})

	view.infoView.SetSelectedFunc(func() {
		view.Pages.HidePage(string(InfoPrompt))
		view.appCtx.App.SetFocus(view.GetLastFocusedView())
	})

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews [][]View) {
	inst.viewNavigation.UpdateOrderedViews(orderedViews, 0)
}

// SetFocusedRow focuses the views in the row of the navigation first, which
// are those in the last row if it isn't called. It is called after the
// navigation is set.
func (inst *ServicePageView) SetFocusedRow(row int) {
	inst.viewNavigation.SetFocusedRow(row)
}

func (inst *ServicePageView) GetViewNavigation() [][]View {
	return inst.viewNavigation.GetOrderedViews()
}

// DisplayMessage shows the message over the page, with the focus on it. It is
// called from the UI goroutine, and from the goroutines that load data, which
// report their errors with it, so the page is only changed on the UI
// goroutine. That is queued from a goroutine of its own, as queuing from the
// UI goroutine would wait on itself. The queued updates can run in any order,
// so each shows the last message of its type, which is the one that stays.
func (inst *ServicePageView) DisplayMessage(messageType MessagePromptType, text string, a ...any) {
	var message = fmt.Sprintf(text, a...)
	var view *MessagePromptView
	switch messageType {
	case InfoPrompt:
		view = inst.infoView
	case ErrorPrompt:
		view = inst.errorView
	default:
		inst.appCtx.Logger.Print(message)
		return
	}

	inst.messagesMtx.Lock()
	inst.lastMessages[messageType] = message
	inst.messagesMtx.Unlock()

	go inst.appCtx.App.QueueUpdateDraw(func() {
		inst.messagesMtx.Lock()
		var last = inst.lastMessages[messageType]
		inst.messagesMtx.Unlock()

		inst.Pages.ShowPage(string(messageType))
		view.SetText(last)
		inst.appCtx.App.SetFocus(view)
	})
}

func (inst *ServicePageView) GetLastFocusedView() tview.Primitive {
	return inst.viewNavigation.GetLastFocusedView()
}

// nestedContent returns the content of a page without its own navigation when
// the content has navigation, such as a ServicePageView added as a tab.
func (inst *ServicePageView) nestedContent() NestedNavigation {
	if len(inst.viewNavigation.GetOrderedViews()) != 0 || inst.MainPage.GetItemCount() != 1 {
		return nil
	}
	if nested, ok := inst.MainPage.GetItem(0).(NestedNavigation); ok {
		return nested
	}
	return nil
}

func (inst *ServicePageView) NavigateWithin(key tcell.Key) bool {
	if nested := inst.nestedContent(); nested != nil {
		return nested.NavigateWithin(key)
	}
	return inst.viewNavigation.NavigateWithin(key)
}

func (inst *ServicePageView) FocusFrom(key tcell.Key) {
	if nested := inst.nestedContent(); nested != nil {
		nested.FocusFrom(key)
		return
	}
	inst.viewNavigation.FocusFrom(key)
}
