package core

import (
	"context"
	"fmt"
	"time"

	"github.com/rivo/tview"
)

type UiDataLoader struct {
	app              *tview.Application
	timeoutSec       time.Duration
	dataLoadComplete chan struct{}
}

func NewUiDataLoader(app *tview.Application, timeoutSec int) UiDataLoader {
	var handler = UiDataLoader{
		app:              app,
		dataLoadComplete: make(chan struct{}),
		timeoutSec:       time.Duration(timeoutSec) * time.Second,
	}

	return handler
}

func (inst *UiDataLoader) AsyncLoadData(handler func()) {
	go func() {
		handler()
		inst.dataLoadComplete <- struct{}{}
	}()
}

func (inst *UiDataLoader) AsyncUpdateView(view View, updateViewFunc func()) {
	go func() {
		var idx = 0
		var originalTitle string
		var titleRead = make(chan struct{})
		inst.app.QueueUpdate(func() {
			originalTitle = view.GetTitle()
			close(titleRead)
		})
		<-titleRead

		var loadingSymbol = [...]string{"⢎⡰", "⢎⡡", "⢎⡑", "⢎⠱", "⠎⡱", "⢊⡱", "⢌⡱", "⢆⡱"}
		var timeoutCtx, cancelFunc = context.WithTimeout(context.Background(), inst.timeoutSec)
		defer cancelFunc()

		for {
			select {
			case <-inst.dataLoadComplete:
				inst.app.QueueUpdateDraw(func() {
					view.SetTitle(originalTitle)
					updateViewFunc()
				})
				return
			case <-timeoutCtx.Done():
				inst.app.QueueUpdateDraw(func() { view.SetTitle(originalTitle + "[Timed out]") })
				return
			default:
				var title = fmt.Sprintf("%s "+originalTitle, loadingSymbol[idx])
				inst.app.QueueUpdateDraw(func() { view.SetTitle(title) })
				idx = (idx + 1) % len(loadingSymbol)
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
}
