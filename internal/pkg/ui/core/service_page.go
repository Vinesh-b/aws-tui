package core

import (
	"fmt"
	"log"

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
	viewNavigation *ViewNavigation
	errorView      *MessagePromptView
	infoView       *MessagePromptView
	lastFocusView  tview.Primitive
	app            *tview.Application
	logger         *log.Logger
}

func NewServicePageView(
	app *tview.Application,
	logger *log.Logger,
) *ServicePageView {
	var flex = tview.NewFlex()
	var viewNav = NewViewNavigation(flex, nil, app)
	viewNav.SetNavigationKeys(
		APP_KEY_BINDINGS.ViewFocusDown,
		APP_KEY_BINDINGS.ViewFocusUp,
	)

	var view = &ServicePageView{
		MainPage:       flex,
		Pages:          tview.NewPages(),
		errorView:      NewMessagePromptView(app),
		infoView:       NewMessagePromptView(app),
		viewNavigation: viewNav,
		app:            app,
		logger:         logger,
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
		view.app.SetFocus(view.GetLastFocusedView())
	})

	view.infoView.SetSelectedFunc(func() {
		view.Pages.HidePage(string(InfoPrompt))
		view.app.SetFocus(view.GetLastFocusedView())
	})

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews []View) {
	inst.viewNavigation.UpdateOrderedViews(orderedViews, 0)
}

func (inst *ServicePageView) GetViewNavigation() []View {
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
		inst.logger.Print(message)
		return
	}

	inst.Pages.ShowPage(string(messageType))
	view.SetText(message)
	inst.app.SetFocus(view)
}

func (inst *ServicePageView) GetLastFocusedView() tview.Primitive {
	return inst.viewNavigation.GetLastFocusedView()
}
