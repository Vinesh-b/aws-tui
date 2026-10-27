package core

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type commandLineTest struct {
	t     *testing.T
	app   *tview.Application
	pages *tview.Pages
	line  *CommandLine
	table *tview.Table
	ran   []string
	on    []tview.Primitive
}

func newCommandLineTest(t *testing.T) *commandLineTest {
	var appCtx = newTestAppContext()
	var test = &commandLineTest{t: t, app: appCtx.App, pages: tview.NewPages(), table: tview.NewTable()}
	test.pages.AddPage("service", test.table, true, true)
	test.line = NewCommandLine(appCtx, test.pages)
	test.line.SetRunFunc(func(command string, focused tview.Primitive) {
		test.ran = append(test.ran, command)
		test.on = append(test.on, focused)
	})
	test.app.SetFocus(test.table)
	return test
}

func (test *commandLineTest) key(event *tcell.EventKey) *tcell.EventKey {
	return test.line.HandleEvent(event, test.app.GetFocus())
}

func (test *commandLineTest) typeColon() *tcell.EventKey {
	return test.key(tcell.NewEventKey(tcell.KeyRune, ':', tcell.ModNone))
}

func (test *commandLineTest) pressInInput(key tcell.Key) {
	test.line.input.InputHandler()(tcell.NewEventKey(key, 0, tcell.ModNone), func(tview.Primitive) {})
}

func (test *commandLineTest) assertShown(want bool) {
	test.t.Helper()
	if shown := slices.Contains(test.pages.GetPageNames(true), COMMAND_LINE_PAGE); shown != want {
		test.t.Errorf("command line shown = %t, want %t", shown, want)
	}
}

func TestCommandLine__ColonOpensItOnTheFocusedView(t *testing.T) {
	var test = newCommandLineTest(t)

	if event := test.typeColon(); event != nil {
		t.Errorf("the colon was passed on as %v", event.Name())
	}

	test.assertShown(true)
	if !test.line.IsOpen() || test.app.GetFocus() != test.line.input {
		t.Error("the input of the command line does not have the focus")
	}
}

func TestCommandLine__EnterRunsTheCommandOnTheViewItWasOpenedOn(t *testing.T) {
	var test = newCommandLineTest(t)
	test.typeColon()
	test.line.input.SetText("w out.csv")

	test.pressInInput(tcell.KeyEnter)

	if !slices.Equal(test.ran, []string{"w out.csv"}) || test.on[0] != test.table {
		t.Errorf("ran %q on %v, want w out.csv on the table", test.ran, test.on)
	}
	test.assertShown(false)
	if test.line.IsOpen() || test.app.GetFocus() != test.table {
		t.Error("the focus did not go back to the table")
	}
}

func TestCommandLine__EscClosesItWithoutRunning(t *testing.T) {
	var test = newCommandLineTest(t)
	test.typeColon()
	test.line.input.SetText("q")

	test.pressInInput(tcell.KeyEscape)

	if len(test.ran) != 0 {
		t.Errorf("ran %q", test.ran)
	}
	test.assertShown(false)
	if test.app.GetFocus() != test.table {
		t.Error("the focus did not go back to the table")
	}
}

func TestCommandLine__EnterWithNothingTypedRunsNothing(t *testing.T) {
	var test = newCommandLineTest(t)
	test.typeColon()

	test.pressInInput(tcell.KeyEnter)

	if len(test.ran) != 0 {
		t.Errorf("ran %q", test.ran)
	}
}

func TestCommandLine__ColonIsTypedIntoTextInputs(t *testing.T) {
	var test = newCommandLineTest(t)
	var input = tview.NewInputField()
	test.app.SetFocus(input)

	if event := test.typeColon(); event == nil {
		t.Fatal("the colon was not passed on to the input")
	}
	test.assertShown(false)
}

func TestCommandLine__AMessageStaysUntilTheNextKey(t *testing.T) {
	var test = newCommandLineTest(t)

	test.line.ShowError("not a command or service: nope")

	test.assertShown(true)
	if test.app.GetFocus() != test.table {
		t.Error("showing the message moved the focus")
	}
	if got := test.line.message.GetText(false); got != "not a command or service: nope" {
		t.Errorf("message = %q", got)
	}

	var j = tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone)
	if event := test.key(j); event != j {
		t.Error("the key that hides the message was not passed on")
	}
	test.assertShown(false)
	if test.app.GetFocus() != test.table {
		t.Error("hiding the message moved the focus")
	}
}

func TestCommandLine__FollowsItsKeyBinding(t *testing.T) {
	keepBindings(t)
	APP_KEY_BINDINGS.CommandLine = ';'
	var test = newCommandLineTest(t)

	if event := test.typeColon(); event == nil {
		t.Error("the colon opened the line, which is on ; now")
	}
	if event := test.key(tcell.NewEventKey(tcell.KeyRune, ';', tcell.ModNone)); event != nil {
		t.Error("; did not open the line")
	}
}
