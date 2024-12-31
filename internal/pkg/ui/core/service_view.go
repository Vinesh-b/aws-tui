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

type ServicePageView struct {
	*tview.Pages
	MainPage       *tview.Flex
	ViewNavigation *ViewNavigation
	errorView      *ErrorMessageView
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
		ViewNavigation: viewNav,
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
	})

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews []View) {
	inst.ViewNavigation.UpdateOrderedViews(orderedViews, 0)
}

func (inst *ServicePageView) SetAndDisplayError(text string) {
	inst.errorView.SetText(text)
	inst.Pages.ShowPage("ERROR")
}
