package core

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newMessageTest is a running app that shows a service page
func newMessageTest(t *testing.T) (*AppContext, *ServicePageView) {
	t.Helper()
	var theme = AppTheme{}
	var appCtx = NewAppContext(context.Background(), tview.NewApplication(), nil, log.New(io.Discard, "", 0), &theme)

	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	var page = NewServicePageView(appCtx)
	page.MainPage.AddItem(tview.NewBox().SetTitle("Orders"), 0, 1, true)
	appCtx.App.SetScreen(screen).SetRoot(page, true)

	var stopped = make(chan struct{})
	go func() {
		defer close(stopped)
		_ = appCtx.App.Run()
	}()
	t.Cleanup(func() {
		appCtx.App.Stop()
		<-stopped
	})
	return appCtx, page
}

// shownMessage is the text of the message of the type, if its prompt is the
// page in front with the focus, read on the UI goroutine
func shownMessage(t *testing.T, appCtx *AppContext, page *ServicePageView, messageType MessagePromptType) string {
	t.Helper()
	var shown = make(chan string, 1)
	appCtx.App.QueueUpdate(func() {
		var view = page.errorView
		if messageType == InfoPrompt {
			view = page.infoView
		}
		var front, _ = page.GetFrontPage()
		if front != string(messageType) || !view.HasFocus() {
			shown <- ""
			return
		}
		shown <- view.textView.GetText(true)
	})
	select {
	case text := <-shown:
		return text
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the UI goroutine")
		return ""
	}
}

// Data is loaded on goroutines of its own, which report their errors there
func TestServicePageView__ErrorOfALoadIsShownWhileTheAppDraws(t *testing.T) {
	var appCtx, page = newMessageTest(t)
	// The app keeps drawing while the error comes, as it does while a page
	// is used
	var stopDrawing = make(chan struct{})
	var drawing = make(chan struct{})
	go func() {
		defer close(drawing)
		for {
			select {
			case <-stopDrawing:
				return
			default:
				appCtx.App.QueueUpdateDraw(func() {})
			}
		}
	}()
	defer func() {
		close(stopDrawing)
		<-drawing
	}()

	var loader = NewUiDataLoader(appCtx, 10)
	loader.AsyncLoadData(func(ctx context.Context) {
		page.DisplayMessage(ErrorPrompt, "%s", "AccessDeniedException: orders")
	})

	waitFor(t, "the error to be shown", func() bool {
		return shownMessage(t, appCtx, page, ErrorPrompt) == "AccessDeniedException: orders"
	})
}

// Messages are shown from the UI goroutine too, such as when a resource to
// open isn't found, which must not wait on itself
func TestServicePageView__MessageFromTheUIGoroutineIsShown(t *testing.T) {
	var appCtx, page = newMessageTest(t)

	appCtx.App.QueueUpdate(func() {
		page.DisplayMessage(InfoPrompt, "Switched to region %s", "eu-west-1")
	})

	waitFor(t, "the message to be shown", func() bool {
		return shownMessage(t, appCtx, page, InfoPrompt) == "Switched to region eu-west-1"
	})
}

func TestServicePageView__LastOfTheMessagesInARowIsShown(t *testing.T) {
	var appCtx, page = newMessageTest(t)

	appCtx.App.QueueUpdate(func() {
		for i := range 20 {
			page.DisplayMessage(ErrorPrompt, "error %d", i)
		}
	})

	waitFor(t, "the last error to be shown", func() bool {
		return shownMessage(t, appCtx, page, ErrorPrompt) == "error 19"
	})
	// None that came before it is shown after it
	var deadline = time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := shownMessage(t, appCtx, page, ErrorPrompt); got != "error 19" {
			t.Fatalf("message = %q after the last, want error 19", got)
		}
	}
}
