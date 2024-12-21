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

type ServiceView struct {
	RootView              *tview.Flex
	LastFocusedView       tview.Primitive
	viewResizeEnabled     bool
	topView               View
	bottomView            View
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

func (inst *ServiceView) InitViewNavigation(orderedViews []View) {
	var viewIdx = 0
	var numViews = len(orderedViews)
	inst.RootView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

		if inst.viewResizeEnabled {
			event = inst.paneResizeHightHandler(event)
		}

		return event
	})
}

func (inst *ServiceView) InitViewTabNavigation(rootView RootView, orderedViews []View) {
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
	topView View, bottomView View,
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
