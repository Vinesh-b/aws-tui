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
	LastFocusedView tview.Primitive
	selectedViewIdx int
	app             *tview.Application
	logger          *log.Logger
}

func NewServicePageView(
	app *tview.Application,
	logger *log.Logger,
) *ServicePageView {
	var view = &ServicePageView{
		Flex:            tview.NewFlex(),
		LastFocusedView: nil,
		app:             app,
		logger:          logger,
	}

	view.SetDirection(tview.FlexRow)

	return view
}

func (inst *ServicePageView) InitViewNavigation(orderedViews []View) {
	var viewIdx = 0
	var numViews = len(orderedViews)
	inst.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlJ:
			viewIdx = (viewIdx - 1 + numViews) % numViews
			inst.LastFocusedView = orderedViews[viewIdx]
			inst.app.SetFocus(inst.LastFocusedView)
			return nil
		case tcell.KeyCtrlK:
			viewIdx = (viewIdx + 1) % numViews
			inst.LastFocusedView = orderedViews[viewIdx]
			inst.app.SetFocus(inst.LastFocusedView)
			return nil
		}

		return event
	})
}
