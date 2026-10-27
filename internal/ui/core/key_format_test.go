package core

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestFormatKey(t *testing.T) {
	var tests = map[tcell.Key]string{
		tcell.KeyESC:   "Esc",
		tcell.KeyPgUp:  "PgUp",
		tcell.KeyPgDn:  "PgDn",
		tcell.KeyCtrlX: "Ctrl+X",
		tcell.KeyCtrlU: "Ctrl+U",
		tcell.KeyEnter: "Enter",
		tcell.KeyUp:    "Up",

		tcell.KeyCtrlSpace: "Ctrl+Space",
		tcell.KeyBackspace: "Backspace",
	}
	for key, want := range tests {
		if got := FormatKey(key); got != want {
			t.Errorf("FormatKey(%v) = %q, want %q", key, got, want)
		}
	}
}

func TestFormatRune(t *testing.T) {
	if got := FormatRune('?'); got != "?" {
		t.Errorf("FormatRune('?') = %q", got)
	}
	if got := FormatRune(' '); got != "Space" {
		t.Errorf("FormatRune(' ') = %q", got)
	}
}

func TestFormatModRune(t *testing.T) {
	if got := FormatModRune(tcell.ModAlt, ']'); got != "Alt+]" {
		t.Errorf("got %q", got)
	}
	if got := FormatModRune(tcell.ModCtrl|tcell.ModShift, 'x'); got != "Ctrl+Shift+x" {
		t.Errorf("got %q", got)
	}
}

func TestFormatModKey(t *testing.T) {
	if got := FormatModKey(tcell.ModAlt, tcell.KeyLeft); got != "Alt+Left" {
		t.Errorf("got %q", got)
	}
}
