package core

import (
	"context"
	"fmt"
	"time"

	"github.com/rivo/tview"
)

// loadingTick is how often the loading symbol of a view moves on
const loadingTick = 100 * time.Millisecond

// UiDataLoader loads data in the background and then updates a view with it.
// The load is given a context that ends when the timeout passes or the app
// shuts down, so a call that takes too long is stopped and not left running.
// A loader is for one load: call AsyncLoadData and AsyncUpdateView once each.
type UiDataLoader struct {
	app              *tview.Application
	appCtx           context.Context
	loadCtx          context.Context
	cancelLoad       context.CancelFunc
	dataLoadComplete chan struct{}
}

func NewUiDataLoader(appCtx *AppContext, timeoutSec int) UiDataLoader {
	return newUiDataLoader(appCtx, time.Duration(timeoutSec)*time.Second)
}

func newUiDataLoader(appCtx *AppContext, timeout time.Duration) UiDataLoader {
	var parent = appCtx.Ctx
	if parent == nil {
		parent = context.Background()
	}
	var loadCtx, cancelLoad = context.WithTimeout(parent, timeout)

	return UiDataLoader{
		app:              appCtx.App,
		appCtx:           parent,
		loadCtx:          loadCtx,
		cancelLoad:       cancelLoad,
		dataLoadComplete: make(chan struct{}),
	}
}

// AsyncLoadData runs the handler in the background. The handler must pass its
// context to the calls it makes, which is what stops them at the timeout.
func (inst *UiDataLoader) AsyncLoadData(handler func(ctx context.Context)) {
	go func() {
		// Closed and not sent, so that the load never waits on a view that
		// has stopped waiting for it
		defer close(inst.dataLoadComplete)
		handler(inst.loadCtx)
	}()
}

// AsyncUpdateView shows the view as loading until the data is loaded and then
// runs updateViewFunc on the UI goroutine. After the timeout the view is shown
// as timed out and updateViewFunc is not run, even if the load ends later.
func (inst *UiDataLoader) AsyncUpdateView(view View, updateViewFunc func()) {
	go func() {
		defer inst.cancelLoad()

		var idx = 0
		var originalTitle string
		var titleRead = make(chan struct{})
		inst.app.QueueUpdate(func() {
			originalTitle = view.GetTitle()
			close(titleRead)
		})
		<-titleRead

		var loadingSymbol = [...]string{"⢎⡰", "⢎⡡", "⢎⡑", "⢎⠱", "⠎⡱", "⢊⡱", "⢌⡱", "⢆⡱"}
		var ticker = time.NewTicker(loadingTick)
		defer ticker.Stop()

		var showLoading = func() {
			var title = fmt.Sprintf("%s "+originalTitle, loadingSymbol[idx])
			inst.app.QueueUpdateDraw(func() { view.SetTitle(title) })
			idx = (idx + 1) % len(loadingSymbol)
		}
		showLoading()

		for {
			select {
			case <-inst.dataLoadComplete:
				inst.app.QueueUpdateDraw(func() {
					view.SetTitle(originalTitle)
					updateViewFunc()
				})
				return
			case <-inst.loadCtx.Done():
				// The app is shutting down, so there is no view left to update
				if inst.appCtx.Err() != nil {
					return
				}
				inst.app.QueueUpdateDraw(func() { view.SetTitle(originalTitle + "[Timed out]") })
				return
			case <-ticker.C:
				showLoading()
			}
		}
	}()
}
