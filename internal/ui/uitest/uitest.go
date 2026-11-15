package uitest

import (
	"context"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableView interface {
	GetCellText(row int, column int) string
	GetTable() *tview.Table
}

// cancels are how the apps of the app contexts made here end their background
// work, by the context each was made with
var (
	cancelsMtx sync.Mutex
	cancels    = map[*tview.Application]context.CancelFunc{}
)

// NewAppContext is an app context where every resource can be opened, as the
// pages of the services that open them are not part of a test of one service
//
// Its context is cancelled when the test stops the app with RunApp, so what
// runs in the background for the app, such as polling the session, stops too.
func NewAppContext() *core.AppContext {
	var ctx, cancel = context.WithCancel(context.Background())
	var appCtx = core.NewAppContext(
		ctx, tview.NewApplication(), nil, log.New(io.Discard, "", 0), &core.AppTheme{},
	)
	cancelsMtx.Lock()
	cancels[appCtx.App] = cancel
	cancelsMtx.Unlock()
	appCtx.SetResourceOpener(nil, func(arn.ARN) bool { return true })
	return appCtx
}

func NewServiceContext[Api any]() *core.ServiceContext[Api] {
	var noApi Api
	return core.NewServiceViewContext(NewAppContext(), noApi)
}

func NewServiceContextWith[Api any](api Api) *core.ServiceContext[Api] {
	return core.NewServiceViewContext(NewAppContext(), api)
}

func AssertDetailsRows(t *testing.T, table *core.DetailsTable, want []core.TableRow) {
	t.Helper()
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if extra := table.GetCellText(len(want), 0); extra != "" {
		t.Errorf("unexpected extra row starting with %q", extra)
	}
}

func AssertTableRows(t *testing.T, table TableView, want []core.TableRow) {
	t.Helper()
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if got := table.GetTable().GetRowCount(); got != len(want) {
		t.Errorf("row count = %d, want %d", got, len(want))
	}
}

func PressEnter(table TableView) {
	var handler = table.GetTable().InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}

func PressTableRune(table TableView, r rune) {
	var handler = table.GetTable().InputHandler()
	handler(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(tview.Primitive) {})
}

const asyncTimeout = 3 * time.Second

// bubbleSettleTime is longer than any timer the app sets, such as the delay
// before the selection of a list loads its row, so that RunApp can let them
// all fire before it stops an app that runs in a bubble
const bubbleSettleTime = time.Minute

// bubbled are the tests running in RunInBubble, by the T it gave them
var bubbled sync.Map

// RunInBubble runs the test in a synctest bubble, where time is fake: it only
// moves on when every goroutine of the test is waiting, and then jumps to the
// next timer. Waiting for a delay, such as the selection of a list settling,
// takes no real time.
//
// Use it rather than synctest.Test for tests that call RunApp, which then lets
// the work left in the background finish before it stops the app. A queued
// update that the stopped app never runs would leave its goroutine waiting
// for ever, which ends the whole test binary in a bubble.
func RunInBubble(t *testing.T, test func(t *testing.T)) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		bubbled.Store(t, struct{}{})
		t.Cleanup(func() { bubbled.Delete(t) })
		test(t)
	})
}

// cancelAppContext ends the background work of the app, if it was made with
// NewAppContext
func cancelAppContext(app *tview.Application) {
	cancelsMtx.Lock()
	defer cancelsMtx.Unlock()
	if cancel, ok := cancels[app]; ok {
		cancel()
		delete(cancels, app)
	}
}

func RunApp(t *testing.T, app *tview.Application) tcell.SimulationScreen {
	t.Helper()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init screen: %v", err)
	}
	app.SetScreen(screen)

	var stopped = make(chan struct{})
	go func() {
		defer close(stopped)
		_ = app.Run()
	}()
	var _, inBubble = bubbled.Load(t)
	t.Cleanup(func() {
		if inBubble {
			cancelAppContext(app)
			// The timers and loads still under way queue updates, which the
			// app has to run for them to end. Fake time passes at once.
			time.Sleep(bubbleSettleTime) // sleep-ok: fake time, in a synctest bubble
		}
		app.Stop()
		<-stopped
		cancelAppContext(app)
	})
	OnUI(t, app, func() {})
	return screen
}

func OnUI(t *testing.T, app *tview.Application, fn func()) {
	t.Helper()
	var done = make(chan struct{})
	app.QueueUpdate(func() {
		fn()
		close(done)
	})
	select {
	case <-done:
	case <-time.After(asyncTimeout):
		t.Fatal("timed out waiting for the UI goroutine")
	}
}

// Eventually waits for the condition to hold. The optional description names
// what was being waited for and is shown when it times out.
func Eventually(t *testing.T, app *tview.Application, condition func() bool, description ...string) {
	t.Helper()
	var deadline = time.Now().Add(asyncTimeout)
	for time.Now().Before(deadline) {
		var ok bool
		OnUI(t, app, func() { ok = condition() })
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond) // sleep-ok: the polling interval
	}
	t.Fatalf("condition not met within %s%s", asyncTimeout, describe(description))
}

// Consistently checks that the condition keeps holding for the whole duration
// and fails as soon as it doesn't. Use it instead of sleeping and then asserting
// that nothing has changed.
func Consistently(
	t *testing.T, app *tview.Application, duration time.Duration, condition func() bool, description ...string,
) {
	t.Helper()
	var deadline = time.Now().Add(duration)
	for {
		var ok bool
		OnUI(t, app, func() { ok = condition() })
		if !ok {
			t.Fatalf("condition stopped holding%s", describe(description))
		}
		if !time.Now().Before(deadline) {
			return
		}
		time.Sleep(10 * time.Millisecond) // sleep-ok: the polling interval
	}
}

func describe(description []string) string {
	if len(description) == 0 {
		return ""
	}
	return ": " + strings.Join(description, " ")
}

func CaptureErrors() (callback func(text string, a ...any), received chan string) {
	received = make(chan string, 8)
	callback = func(text string, a ...any) { received <- fmt.Sprintf(text, a...) }
	return callback, received
}

func AwaitError(t *testing.T, received <-chan string) string {
	t.Helper()
	select {
	case message := <-received:
		return message
	case <-time.After(asyncTimeout):
		t.Fatal("error callback was not called")
		return ""
	}
}

func AwaitRows(t *testing.T, app *tview.Application, table TableView, want []core.TableRow) {
	t.Helper()
	Eventually(t, app, func() bool { return table.GetTable().GetRowCount() == len(want) })
	OnUI(t, app, func() { AssertTableRows(t, table, want) })
}

func AwaitDetailsRows(t *testing.T, app *tview.Application, table *core.DetailsTable, want []core.TableRow) {
	t.Helper()
	Eventually(t, app, func() bool { return table.GetCellText(0, 0) == want[0][0] && table.GetCellText(len(want), 0) == "" })
	OnUI(t, app, func() { AssertDetailsRows(t, table, want) })
}

func ScreenText(screen tcell.SimulationScreen) string {
	var cells, width, height = screen.GetContents()
	var text strings.Builder
	for row := range height {
		for col := range width {
			text.WriteString(string(cells[row*width+col].Runes))
		}
		text.WriteString("\n")
	}
	return text.String()
}

func ShowPage(t *testing.T, app *tview.Application, page core.ServicePage) tcell.SimulationScreen {
	t.Helper()
	var screen = RunApp(t, app)
	OnUI(t, app, func() {
		app.SetRoot(page, true)
		app.SetFocus(page.GetLastFocusedView())
	})
	return screen
}

func PressRune(t *testing.T, app *tview.Application, r rune) {
	t.Helper()
	OnUI(t, app, func() {
		var focused = app.GetFocus()
		var handler = focused.InputHandler()
		handler(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(p tview.Primitive) { app.SetFocus(p) })
	})
}

func AwaitScreenText(t *testing.T, app *tview.Application, screen tcell.SimulationScreen, want string) {
	t.Helper()
	var deadline = time.Now().Add(asyncTimeout)
	for time.Now().Before(deadline) {
		var text string
		OnUI(t, app, func() {
			text = ScreenText(screen)
		})
		if strings.Contains(text, want) {
			return
		}
		time.Sleep(10 * time.Millisecond) // sleep-ok: the polling interval
	}
	t.Fatalf("screen never showed %q", want)
}

// PressKey sends the key through the whole focus chain, so key handlers on
// parent views run as they would for a real key press.
func PressKey(t *testing.T, app *tview.Application, key tcell.Key) {
	t.Helper()
	app.QueueEvent(tcell.NewEventKey(key, 0, tcell.ModNone))
	OnUI(t, app, func() {})
}

// DetailsRowOf is the first row of the table with the label, e.g. the heading
// of a section, or -1 if there is none.
func DetailsRowOf(table *core.DetailsTable, label string) int {
	for row := 0; table.GetCellText(row, 0) != ""; row++ {
		if table.GetCellText(row, 0) == label {
			return row
		}
	}
	return -1
}

// MoveThrough is a quick move of the selection of a list from the second row
// below its headings to the fourth, for AssertMovingThroughLoadsOnlyWhereItStops
type MoveThrough[T comparable] struct {
	// Table is the list, with at least four rows below its headings
	Table TableView
	// Passed is the item of the third row, which the selection passes through,
	// and StopsOn the item of the fourth, where it stops
	Passed, StopsOn T
	// Loaded gives every item the api was asked to load so far, oldest first
	Loaded func() []T
	// StraightAway, if set, checks what follows the selection without waiting
	// for it to settle. It runs on the UI goroutine as soon as the selection
	// stops, before the item it stops on can be loaded. It reports with
	// t.Errorf, as t.Fatalf would end the UI goroutine and hang the test.
	StraightAway func()
}

// AssertMovingThroughLoadsOnlyWhereItStops moves the selection quickly
// through the rows and checks that the item it stops on is loaded, last, and
// that the item it passes through never is. The second row may be loaded, as
// the first move after a quiet spell loads straight away.
//
// Run it in RunInBubble, so that waiting for the selection to settle
// takes no real time.
func AssertMovingThroughLoadsOnlyWhereItStops[T comparable](
	t *testing.T, app *tview.Application, move MoveThrough[T],
) {
	t.Helper()
	var before = len(move.Loaded())
	OnUI(t, app, func() {
		for row := 2; row <= 4; row++ {
			move.Table.GetTable().Select(row, 0)
		}
		if move.StraightAway != nil {
			move.StraightAway()
		}
	})

	Eventually(t, app, func() bool {
		return slices.Contains(move.Loaded()[before:], move.StopsOn)
	}, fmt.Sprintf("%v, where the selection stopped, to be loaded", move.StopsOn))
	Consistently(t, app, 2*core.SelectionLoadDelay, func() bool {
		var loaded = move.Loaded()[before:]
		return !slices.Contains(loaded, move.Passed) && loaded[len(loaded)-1] == move.StopsOn
	}, fmt.Sprintf("%v, which the selection passed through, never to be loaded, and %v to be loaded last",
		move.Passed, move.StopsOn))
}

// AwaitDetailsTag waits for the tags section of the table to have the tag
func AwaitDetailsTag(t *testing.T, app *tview.Application, table *core.DetailsTable, key string, value string) {
	t.Helper()
	Eventually(t, app, func() bool {
		var section = DetailsRowOf(table, "Tags")
		if section < 0 {
			return false
		}
		for row := section + 1; table.GetCellText(row, 0) != ""; row++ {
			// Rows in a section are indented
			if strings.TrimSpace(table.GetCellText(row, 0)) == key && table.GetCellText(row, 1) == value {
				return true
			}
		}
		return false
	})
}
