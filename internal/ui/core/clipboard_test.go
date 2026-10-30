package core

import (
	"errors"
	"strings"
	"testing"

	"aws-tui/internal/safetest"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newClipboardTest is an app context drawn on a simulated terminal, which
// copies to a list in place of the clipboard of the system
func newClipboardTest(t *testing.T) (*AppContext, tcell.SimulationScreen, *safetest.Recorder[string]) {
	t.Helper()
	var appCtx = newTestAppContext()
	var screen = tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	appCtx.SetTerminal(screen)
	var copied = &safetest.Recorder[string]{}
	appCtx.SetSystemClipboard(func(text string) error {
		copied.Record(text)
		return nil
	})
	return appCtx, screen, copied
}

func TestCopyText__GoesToTheTerminalAndTheSystem(t *testing.T) {
	var appCtx, screen, copied = newClipboardTest(t)

	appCtx.CopyText("arn:aws:lambda:us-east-1:123456789012:function:orders")

	if got := string(screen.GetClipboardData()); got != "arn:aws:lambda:us-east-1:123456789012:function:orders" {
		t.Errorf("terminal clipboard = %q", got)
	}
	if got := copied.Calls(); len(got) != 1 || got[0] != string(screen.GetClipboardData()) {
		t.Errorf("system clipboard = %v", got)
	}
}

func TestCopyText__WithoutATerminalGoesToTheSystem(t *testing.T) {
	var appCtx = newTestAppContext()
	var copied safetest.Recorder[string]
	appCtx.SetSystemClipboard(func(text string) error {
		copied.Record(text)
		return nil
	})

	appCtx.CopyText("orders")

	if got := copied.Calls(); len(got) != 1 || got[0] != "orders" {
		t.Errorf("system clipboard = %v", got)
	}
}

func TestCopyText__TheSystemFailingIsLoggedOnceAndTheTerminalStillGetsTheText(t *testing.T) {
	var appCtx, screen, _ = newClipboardTest(t)
	var logged safetest.LogCapture
	appCtx.Logger = logged.Logger()
	appCtx.SetSystemClipboard(func(string) error { return errors.New("no xclip") })

	appCtx.CopyText("one")
	appCtx.CopyText("two")

	if got := string(screen.GetClipboardData()); got != "two" {
		t.Errorf("terminal clipboard = %q, want two", got)
	}
	if got := strings.Count(logged.String(), "no xclip"); got != 1 {
		t.Errorf("logged %d times, want once: %q", got, logged.String())
	}
}

func TestSelectableTable__CopyKeyCopiesTheCell(t *testing.T) {
	var appCtx, screen, _ = newClipboardTest(t)
	var table = NewSelectableTable[string]("Functions", TableRow{"Name", "Runtime"}, appCtx)
	table.SetData([]TableRow{{"orders", "go1.x"}}, nil, 0)
	table.Select(1, 1)

	table.table.InputHandler()(tcell.NewEventKey(tcell.KeyRune, APP_KEY_BINDINGS.TextCopy, tcell.ModNone), func(tview.Primitive) {})

	if got := string(screen.GetClipboardData()); got != "go1.x" {
		t.Errorf("copied %q, want the selected cell", got)
	}
}

func TestDetailsTable__CopyKeyCopiesTheValue(t *testing.T) {
	var appCtx, screen, _ = newClipboardTest(t)
	var details = NewDetailsTable("details", appCtx)
	details.SetData([]TableRow{{"Name", "orders"}})
	details.Select(0, 0)

	details.table.InputHandler()(tcell.NewEventKey(tcell.KeyRune, APP_KEY_BINDINGS.TextCopy, tcell.ModNone), func(tview.Primitive) {})

	if got := string(screen.GetClipboardData()); got != "orders" {
		t.Errorf("copied %q, want the value of the row", got)
	}
}

func TestTextArea__CopiesTheSelectedText(t *testing.T) {
	var appCtx, screen, _ = newClipboardTest(t)
	var area = NewTextArea("Payload", appCtx)
	area.SetText(`{"id": 1}`, false)
	area.Select(0, 4)

	area.InputHandler()(tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModCtrl), func(tview.Primitive) {})

	if got := string(screen.GetClipboardData()); got != `{"id` {
		t.Errorf("copied %q, want the selected text", got)
	}
}
