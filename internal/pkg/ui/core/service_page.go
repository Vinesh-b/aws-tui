package core

import (
	"fmt"

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
	lastFocusView  tview.Primitive
	appCtx         *AppContext
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

func (inst *ServicePageView) GetViewNavigation() [][]View {
	return inst.viewNavigation.GetOrderedViews()
}

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

	inst.Pages.ShowPage(string(messageType))
	view.SetText(message)
	inst.appCtx.App.SetFocus(view)
}

func (inst *ServicePageView) GetLastFocusedView() tview.Primitive {
	return inst.viewNavigation.GetLastFocusedView()
}
