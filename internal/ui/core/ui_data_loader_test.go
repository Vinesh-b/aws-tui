package core

import (
	"context"
	"errors"
	"io"
	"log"
	"runtime"
	"testing"
	"time"

	"aws-tui/internal/safetest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newLoaderTest is a running app that shows one view titled "Orders". The
// cancel func shuts the app context down as closing the app does.
func newLoaderTest(t *testing.T) (*AppContext, *tview.Box, context.CancelFunc) {
	t.Helper()
	var ctx, cancel = context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var theme = AppTheme{}
	var appCtx = NewAppContext(ctx, tview.NewApplication(), nil, log.New(io.Discard, "", 0), &theme)

	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	var view = tview.NewBox().SetTitle("Orders")
	appCtx.App.SetScreen(screen).SetRoot(view, true)

	var stopped = make(chan struct{})
	go func() {
		defer close(stopped)
		_ = appCtx.App.Run()
	}()
	t.Cleanup(func() {
		appCtx.App.Stop()
		<-stopped
	})
	// The app starts goroutines of its own, which the tests must not count
	titleOf(t, appCtx, view)
	return appCtx, view, cancel
}

// titleOf reads the title of the view on the UI goroutine
func titleOf(t *testing.T, appCtx *AppContext, view *tview.Box) string {
	t.Helper()
	var title = make(chan string, 1)
	appCtx.App.QueueUpdate(func() { title <- view.GetTitle() })
	select {
	case got := <-title:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the UI goroutine")
		return ""
	}
}

func TestUiDataLoader__UpdatesTheViewOnceTheDataIsLoaded(t *testing.T) {
	var appCtx, view, _ = newLoaderTest(t)
	var loader = newUiDataLoader(appCtx, time.Minute)
	var loadErr safetest.Value[error]
	var updated safetest.Recorder[string]

	loader.AsyncLoadData(func(ctx context.Context) { loadErr.Set(ctx.Err()) })
	loader.AsyncUpdateView(view, func() { updated.Record("orders") })

	waitFor(t, "the view to be updated", func() bool { return updated.Count() == 1 })
	if err := loadErr.Get(); err != nil {
		t.Errorf("the context of the load had ended: %v", err)
	}
	if got := titleOf(t, appCtx, view); got != "Orders" {
		t.Errorf("title = %q, want it as it was before the load", got)
	}
}

// A call that takes longer than the timeout is stopped through its context,
// and what it loaded is not shown once the view said that it timed out
func TestUiDataLoader__TimeoutEndsTheLoadAndTheViewIsNotUpdated(t *testing.T) {
	var appCtx, view, _ = newLoaderTest(t)
	var loader = newUiDataLoader(appCtx, 20*time.Millisecond)
	var loadErr safetest.Value[error]
	var loadEnded = make(chan struct{})
	var updated safetest.Recorder[string]

	loader.AsyncLoadData(func(ctx context.Context) {
		defer close(loadEnded)
		<-ctx.Done()
		loadErr.Set(ctx.Err())
	})
	loader.AsyncUpdateView(view, func() { updated.Record("orders") })

	select {
	case <-loadEnded:
	case <-time.After(2 * time.Second):
		t.Fatal("the load was not stopped at the timeout")
	}
	if err := loadErr.Get(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the load ended with %v, want the deadline", err)
	}
	waitFor(t, "the view to show the timeout", func() bool {
		return titleOf(t, appCtx, view) == "Orders[Timed out]"
	})
	if updated.Count() != 0 {
		t.Error("the view was updated with a load that timed out")
	}
}

// The view stops waiting at the timeout, and a load that ends after that must
// not be left waiting to hand over its data
func TestUiDataLoader__LoadThatOutlivesTheTimeoutIsNotLeftWaiting(t *testing.T) {
	var appCtx, view, _ = newLoaderTest(t)
	var before = runtime.NumGoroutine()
	var loader = newUiDataLoader(appCtx, 20*time.Millisecond)
	var release = make(chan struct{})

	// The load ignores its context, as code that can't be cancelled does
	loader.AsyncLoadData(func(context.Context) { <-release })
	loader.AsyncUpdateView(view, func() {})
	waitFor(t, "the view to show the timeout", func() bool {
		return titleOf(t, appCtx, view) == "Orders[Timed out]"
	})

	close(release)

	waitFor(t, "the load to end", func() bool { return runtime.NumGoroutine() <= before })
}

func TestUiDataLoader__LoadWithoutAViewIsNotLeftWaiting(t *testing.T) {
	var appCtx, _, _ = newLoaderTest(t)
	var before = runtime.NumGoroutine()
	var loader = newUiDataLoader(appCtx, time.Minute)

	loader.AsyncLoadData(func(context.Context) {})

	waitFor(t, "the load to end", func() bool { return runtime.NumGoroutine() <= before })
}

func TestUiDataLoader__ClosingTheAppEndsTheLoad(t *testing.T) {
	var appCtx, view, closeApp = newLoaderTest(t)
	var loader = newUiDataLoader(appCtx, time.Minute)
	var started = make(chan struct{})
	var loadErr = make(chan error, 1)
	var updated safetest.Recorder[string]

	loader.AsyncLoadData(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		loadErr <- ctx.Err()
	})
	loader.AsyncUpdateView(view, func() { updated.Record("orders") })
	<-started

	closeApp()

	select {
	case err := <-loadErr:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the load ended with %v, want cancelled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the load was not stopped when the app closed")
	}
	if updated.Count() != 0 {
		t.Error("the view was updated after the app closed")
	}
}
