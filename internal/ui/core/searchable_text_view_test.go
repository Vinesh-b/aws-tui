package core

import (
	"regexp"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestMatchLines__MatchesCountRuneForEveryMatch(t *testing.T) {
	var texts = []string{
		"",
		"foo",
		"foo\nbar foo\n\nfoo",
		"héllo\nwörld foo\n日本語 foo\nfoo foo",
		"\nfoo\n",
	}
	var expr = regexp.MustCompile("foo")

	for _, text := range texts {
		var positions = expr.FindAllStringIndex(text, -1)
		var lines = matchLines(text, positions)

		if len(lines) != len(positions) {
			t.Fatalf("%q: got %d lines for %d matches", text, len(lines), len(positions))
		}
		for i, pos := range positions {
			var want = countRune(text, '\n', pos[0])
			if lines[i] != want {
				t.Errorf("%q match %d at %d: got %d, want %d", text, i, pos[0], lines[i], want)
			}
		}
	}
}

func TestSearchableTextView__VisualModeFollowsItsKeyBinding(t *testing.T) {
	var original = APP_KEY_BINDINGS
	t.Cleanup(func() { APP_KEY_BINDINGS = original })
	APP_KEY_BINDINGS.TextViewVisualMode = 'V'
	var view = NewSearchableTextView("Record", newTestAppContext())
	var press = func(r rune) {
		view.textArea.InputHandler()(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(tview.Primitive) {})
	}

	press('v')
	if view.vimMode != VimModeNormal {
		t.Fatalf("v changed the mode to %v, want it to stay normal", view.vimMode)
	}
	press('V')
	if view.vimMode != VimModeVisual {
		t.Errorf("V gave mode %v, want visual", view.vimMode)
	}
}

func TestSearchableTextView__InsertKeyDoesNotMakeItEditable(t *testing.T) {
	var view = NewSearchableTextView("Record", newTestAppContext())
	view.SetText("abc", false)
	var press = func(r rune) {
		view.textArea.InputHandler()(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone), func(tview.Primitive) {})
	}

	press(APP_KEY_BINDINGS.TextViewInsertMode)
	press('x')
	if view.vimMode != VimModeNormal {
		t.Errorf("insert key gave mode %v, want normal", view.vimMode)
	}
	if got := view.GetText(); got != "abc" {
		t.Errorf("text is %q after typing, want it unchanged", got)
	}
}
