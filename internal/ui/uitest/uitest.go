package uitest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableView interface {
	GetCellText(row int, column int) string
	GetTable() *tview.Table
}

func NewAppContext() *core.AppContext {
	return core.NewAppContext(
		context.Background(), tview.NewApplication(), nil, nil, &core.AppTheme{},
	)
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

const asyncTimeout = 3 * time.Second

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
	t.Cleanup(func() {
		app.Stop()
		<-stopped
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

func Eventually(t *testing.T, app *tview.Application, condition func() bool) {
	t.Helper()
	var deadline = time.Now().Add(asyncTimeout)
	for time.Now().Before(deadline) {
		var ok bool
		OnUI(t, app, func() { ok = condition() })
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
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
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("screen never showed %q", want)
}
