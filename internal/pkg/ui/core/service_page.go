package core

import (
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
}

type ServicePage interface {
	tview.Primitive
	GetLastFocusedView() tview.Primitive
}

type ServicePageView struct {
	*tview.Pages
	MainPage       *tview.Flex
	viewNavigation *ViewNavigation
	errorView      *ErrorMessageView
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
	viewNav.SetNavigationKeys(tcell.KeyCtrlJ, tcell.KeyCtrlK)

	var view = &ServicePageView{
		MainPage:       flex,
		Pages:          tview.NewPages(),
		errorView:      NewErrorMessageView(app),
		viewNavigation: viewNav,
		app:            app,
		logger:         logger,
	}

	view.MainPage.SetDirection(tview.FlexRow)

	var floatingErrorView = FloatingView("Error", view.errorView, 80, 15)
	view.Pages.
		AddPage("MAIN_PAGE", view.MainPage, true, true).
		AddPage("ERROR", floatingErrorView, true, false)

	view.errorView.SetSelectedFunc(func() {
		view.Pages.HidePage("ERROR")
		view.app.SetFocus(view.GetLastFocusedView())
	})

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews []View) {
	inst.viewNavigation.UpdateOrderedViews(orderedViews, 0)
}

func (inst *ServicePageView) SetAndDisplayError(text string) {
	inst.errorView.SetText(text)
	inst.Pages.ShowPage("ERROR")
	inst.app.SetFocus(inst.errorView)
}

func (inst *ServicePageView) GetLastFocusedView() tview.Primitive {
	return inst.viewNavigation.GetLastFocusedView()
}
