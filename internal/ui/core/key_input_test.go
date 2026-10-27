package core

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestNormalizeKeyEvent__CtrlSpaceIsTheSameKeyFromEveryTerminal(t *testing.T) {
	// The newer keyboard protocols send a space with Ctrl, the others a NUL
	var events = []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModCtrl),
		tcell.NewEventKey(tcell.KeyCtrlSpace, 0, tcell.ModCtrl),
	}
	for _, event := range events {
		if got := NormalizeKeyEvent(event).Key(); got != tcell.KeyCtrlSpace {
			t.Errorf("%v gives %v, want Ctrl+Space", event.Name(), got)
		}
	}
}

func TestNormalizeKeyEvent__LeavesOtherKeysAlone(t *testing.T) {
	var events = []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone),
		tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModCtrl),
		tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
	}
	for _, event := range events {
		if got := NormalizeKeyEvent(event); got != event {
			t.Errorf("%v was changed to %v", event.Name(), got.Name())
		}
	}
}

func TestPaneKey__BackspaceMovesLeftAsCtrlHOutsideOfTextInputs(t *testing.T) {
	var backspace = tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)

	if got := paneKey(backspace, tcell.KeyCtrlH, tview.NewTable()); got != tcell.KeyCtrlH {
		t.Errorf("backspace in a table = %v, want Ctrl+H", got)
	}
	if got := paneKey(backspace, tcell.KeyCtrlH, tview.NewInputField()); got != tcell.KeyBackspace {
		t.Errorf("backspace in an input = %v, want backspace", got)
	}
	if got := paneKey(backspace, tcell.KeyCtrlB, tview.NewTable()); got != tcell.KeyBackspace {
		t.Errorf("backspace with left on Ctrl+B = %v, want backspace", got)
	}
}
