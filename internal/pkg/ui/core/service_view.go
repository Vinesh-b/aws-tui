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
	*tview.Flex
	ViewNavigation *ViewNavigation
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
		Flex:           flex,
		ViewNavigation: viewNav,
		app:            app,
		logger:         logger,
	}

	view.SetDirection(tview.FlexRow)

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews []View) {
	inst.ViewNavigation.UpdateOrderedViews(orderedViews, 0)
}
